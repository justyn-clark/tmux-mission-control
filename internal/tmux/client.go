package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/justynclarknetwork/tmux-mission-control/internal/manifest"
	"github.com/justynclarknetwork/tmux-mission-control/internal/runtime"
)

type Client struct{}

var (
	ErrTmuxNotRunning   = errors.New("tmux server is not running")
	ErrTmuxInaccessible = errors.New("tmux socket is inaccessible")
)

type SessionSummary struct {
	Name         string `json:"name"`
	Windows      int    `json:"windows"`
	Attached     int    `json:"attached"`
	Managed      bool   `json:"managed"`
	Project      string `json:"project,omitempty"`
	ManifestPath string `json:"manifest_path,omitempty"`
}

type SessionMetadata struct {
	Managed      bool
	Project      string
	ManifestPath string
}

type PaneRecord struct {
	PaneID     string
	Window     string
	PaneTitle  string
	CWD        string
	Command    string
	Role       string
	StateFile  string
	Dead       bool
	DeadStatus string
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Apply(plan *runtime.Plan) error {
	if err := c.Preflight(plan); err != nil {
		return err
	}
	sessionCreated := false
	for _, action := range plan.Actions {
		runAction := func() error {
			switch action.Kind {
			case "check-session":
				return nil
			case "startup-hook", "window-startup-hook":
				if err := run(action.Command); err != nil {
					return fmt.Errorf("%s failed: %w", action.Note, err)
				}
				return nil
			default:
				if err := run(action.Command); err != nil {
					return fmt.Errorf("%s: %w", action.Note, err)
				}
				return nil
			}
		}

		if err := runAction(); err != nil {
			if sessionCreated {
				_ = run([]string{"tmux", "kill-session", "-t", plan.SessionName})
			}
			return err
		}
		if action.Kind == "new-session" {
			sessionCreated = true
		}
	}
	return nil
}

func (c *Client) Preflight(plan *runtime.Plan) error {
	if plan == nil {
		return errors.New("tmux plan is nil")
	}
	if plan.SessionName == "" {
		return errors.New("tmux plan session name is empty")
	}
	if _, err := exec.LookPath(tmuxBinary()); err != nil {
		return fmt.Errorf("tmux binary unavailable: %w", err)
	}
	if err := c.ensureSessionMissing(plan.SessionName); err != nil {
		return err
	}
	for _, action := range plan.Actions {
		switch action.Kind {
		case "check-session":
			continue
		case "startup-hook", "window-startup-hook", "new-session", "new-window", "split-window", "send-keys":
			if strings.TrimSpace(action.CWD) == "" {
				return fmt.Errorf("%s preflight failed: cwd is empty", action.Note)
			}
			info, err := os.Stat(action.CWD)
			if err != nil {
				return fmt.Errorf("%s preflight failed: %w", action.Note, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("%s preflight failed: cwd is not a directory: %s", action.Note, action.CWD)
			}
		}
	}
	return nil
}

func (c *Client) StopSession(session string) error {
	managedValue, managedErr := c.sessionOption(session, "@tmc_managed")
	if managedErr != nil && IsTmuxInaccessible(managedErr) {
		return managedErr
	}

	manifestPath, _ := c.sessionOption(session, "@tmc_manifest")
	if managedValue == "1" {
		if strings.TrimSpace(manifestPath) == "" {
			return fmt.Errorf("managed session %q is missing @tmc_manifest metadata; refusing to stop because shutdown hooks cannot be audited", session)
		}
		m, err := manifest.LoadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("managed session %q recorded unreadable shutdown hook manifest %q: %w", session, manifestPath, err)
		}
		for i := len(m.Windows) - 1; i >= 0; i-- {
			window := m.Windows[i]
			for j := len(window.ShutdownHooks) - 1; j >= 0; j-- {
				hook := window.ShutdownHooks[j]
				if err := runHook(firstNonEmpty(hook.CWD, window.Root), mergeEnv(m.Env, window.Env, hook.Env), hook.Command); err != nil {
					return fmt.Errorf("window shutdown hook %s: %w", window.Name, err)
				}
			}
		}
		for i := len(m.ShutdownHooks) - 1; i >= 0; i-- {
			hook := m.ShutdownHooks[i]
			if err := runHook(firstNonEmpty(hook.CWD, m.Root), mergeEnv(m.Env, hook.Env), hook.Command); err != nil {
				return fmt.Errorf("project shutdown hook: %w", err)
			}
		}
	}
	if err := run([]string{"tmux", "kill-session", "-t", session}); err != nil {
		return err
	}
	return nil
}

func (c *Client) ListSessions() ([]SessionSummary, error) {
	output, err := runOutput([]string{"tmux", "list-sessions", "-F", "#{session_name}\t#{session_windows}\t#{session_attached}"})
	if err != nil {
		classified := classifyTmuxError(err)
		if errors.Is(classified, ErrTmuxNotRunning) {
			return nil, classified
		}
		return nil, classified
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	sessions := make([]SessionSummary, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 3 {
			continue
		}
		windows, _ := strconv.Atoi(parts[1])
		attached, _ := strconv.Atoi(parts[2])
		metadata, _ := c.SessionMetadata(parts[0])
		sessions = append(sessions, SessionSummary{
			Name:         parts[0],
			Windows:      windows,
			Attached:     attached,
			Managed:      metadata.Managed,
			Project:      metadata.Project,
			ManifestPath: metadata.ManifestPath,
		})
	}
	return sessions, nil
}

func (c *Client) SessionExists(session string) (bool, error) {
	err := run([]string{"tmux", "has-session", "-t", session})
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "can't find session") {
		return false, nil
	}
	classified := classifyTmuxError(err)
	return false, classified
}

func (c *Client) WindowCount(session string) (int, error) {
	output, err := runOutput([]string{"tmux", "display-message", "-p", "-t", session, "#{session_windows}"})
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (c *Client) ListPanes(session string) ([]PaneRecord, error) {
	format := "#{pane_id}\t#{window_name}\t#{pane_title}\t#{pane_current_path}\t#{pane_dead}\t#{?pane_dead_status,#{pane_dead_status},-}"
	output, err := runOutput([]string{"tmux", "list-panes", "-t", session, "-F", format})
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	panes := make([]PaneRecord, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 6 {
			continue
		}
		command, _ := c.paneOption(parts[0], "@tmc_command")
		role, _ := c.paneOption(parts[0], "@tmc_role")
		stateFile, _ := c.paneOption(parts[0], "@tmc_state_file")
		panes = append(panes, PaneRecord{
			PaneID:     parts[0],
			Window:     parts[1],
			PaneTitle:  parts[2],
			CWD:        parts[3],
			Command:    command,
			Role:       role,
			StateFile:  stateFile,
			Dead:       parts[4] == "1",
			DeadStatus: parts[5],
		})
	}
	return panes, nil
}

func (c *Client) SessionMetadata(session string) (SessionMetadata, error) {
	managed, err := c.sessionOption(session, "@tmc_managed")
	if err != nil {
		classified := classifyTmuxError(err)
		if errors.Is(classified, ErrTmuxInaccessible) {
			return SessionMetadata{}, classified
		}
	}
	project, _ := c.sessionOption(session, "@tmc_project")
	manifestPath, _ := c.sessionOption(session, "@tmc_manifest")
	return SessionMetadata{
		Managed:      managed == "1",
		Project:      project,
		ManifestPath: manifestPath,
	}, nil
}

func (c *Client) sessionOption(session, option string) (string, error) {
	output, err := runOutput([]string{"tmux", "show-options", "-v", "-t", session, option})
	if err != nil {
		return "", classifyTmuxError(err)
	}
	return output, nil
}

func (c *Client) paneOption(paneID, option string) (string, error) {
	return runOutput([]string{"tmux", "show-options", "-p", "-v", "-t", paneID, option})
}

func (c *Client) ensureSessionMissing(session string) error {
	exists, err := c.SessionExists(session)
	if err != nil {
		if IsTmuxNotRunning(err) {
			return nil
		}
		return err
	}
	if exists {
		return fmt.Errorf("session %q already exists", session)
	}
	return nil
}

func run(args []string) error {
	name, commandArgs := commandArgs(args)
	cmd := exec.Command(name, commandArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return err
		}
		return errors.New(text)
	}
	return nil
}

func runOutput(args []string) (string, error) {
	name, commandArgs := commandArgs(args)
	cmd := exec.Command(name, commandArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return "", err
		}
		return "", errors.New(text)
	}
	return strings.TrimSpace(string(output)), nil
}

func commandArgs(args []string) (string, []string) {
	if len(args) == 0 {
		return "", nil
	}
	if args[0] != "tmux" {
		return args[0], args[1:]
	}
	command := append([]string{}, args[1:]...)
	if socket := os.Getenv("TMC_TMUX_SOCKET"); strings.TrimSpace(socket) != "" {
		command = append([]string{"-S", socket}, command...)
	}
	return tmuxBinary(), command
}

func tmuxBinary() string {
	if value := os.Getenv("TMC_TMUX_BIN"); strings.TrimSpace(value) != "" {
		return value
	}
	return "tmux"
}

func classifyTmuxError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "failed to connect"), strings.Contains(message, "No such file or directory"), strings.Contains(message, "no server running"):
		return fmt.Errorf("%w: %s", ErrTmuxNotRunning, message)
	case strings.Contains(message, "Operation not permitted"), strings.Contains(strings.ToLower(message), "permission denied"):
		return fmt.Errorf("%w: %s", ErrTmuxInaccessible, message)
	default:
		return err
	}
}

func IsTmuxNotRunning(err error) bool {
	return errors.Is(err, ErrTmuxNotRunning)
}

func IsTmuxInaccessible(err error) bool {
	return errors.Is(err, ErrTmuxInaccessible)
}

func runHook(cwd string, env map[string]string, command string) error {
	return run([]string{"/bin/sh", "-lc", buildShellCommand(cwd, env, command)})
}

func buildShellCommand(cwd string, env map[string]string, command string) string {
	parts := []string{fmt.Sprintf("cd %s", shellQuote(cwd))}
	for _, key := range sortedKeys(env) {
		parts = append(parts, fmt.Sprintf("export %s=%s", key, shellQuote(env[key])))
	}
	parts = append(parts, command)
	return strings.Join(parts, "; ")
}

func mergeEnv(envs ...map[string]string) map[string]string {
	result := map[string]string{}
	for _, env := range envs {
		for key, value := range env {
			result[key] = value
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
