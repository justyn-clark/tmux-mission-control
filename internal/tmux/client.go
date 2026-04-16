package tmux

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/justynclarknetwork/tmux-mission-control/internal/manifest"
	"github.com/justynclarknetwork/tmux-mission-control/internal/runtime"
)

type Client struct{}

type SessionSummary struct {
	Name     string
	Windows  int
	Attached int
	Managed  bool
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
	for _, action := range plan.Actions {
		switch action.Kind {
		case "check-session":
			if err := c.ensureSessionMissing(plan.SessionName); err != nil {
				return err
			}
			continue
		case "startup-hook", "window-startup-hook":
			if err := run(action.Command); err != nil {
				return fmt.Errorf("%s failed: %w", action.Note, err)
			}
			continue
		default:
			if err := run(action.Command); err != nil {
				return fmt.Errorf("%s: %w", action.Note, err)
			}
		}
	}
	return nil
}

func (c *Client) StopSession(session string) error {
	var hookErr error
	manifestPath, _ := c.sessionOption(session, "@tmc_manifest")
	if manifestPath != "" {
		m, err := manifest.LoadFile(manifestPath)
		if err == nil {
			for i := len(m.Windows) - 1; i >= 0; i-- {
				window := m.Windows[i]
				for j := len(window.ShutdownHooks) - 1; j >= 0; j-- {
					hook := window.ShutdownHooks[j]
					if err := runHook(firstNonEmpty(hook.CWD, window.Root), mergeEnv(m.Env, window.Env, hook.Env), hook.Command); err != nil && hookErr == nil {
						hookErr = fmt.Errorf("window shutdown hook %s: %w", window.Name, err)
					}
				}
			}
			for i := len(m.ShutdownHooks) - 1; i >= 0; i-- {
				hook := m.ShutdownHooks[i]
				if err := runHook(firstNonEmpty(hook.CWD, m.Root), mergeEnv(m.Env, hook.Env), hook.Command); err != nil && hookErr == nil {
					hookErr = fmt.Errorf("project shutdown hook: %w", err)
				}
			}
		}
	}
	if err := run([]string{"tmux", "kill-session", "-t", session}); err != nil {
		return err
	}
	return hookErr
}

func (c *Client) ListSessions() ([]SessionSummary, error) {
	output, err := runOutput([]string{"tmux", "list-sessions", "-F", "#{session_name}\t#{session_windows}\t#{session_attached}"})
	if err != nil {
		if strings.Contains(err.Error(), "failed to connect") {
			return nil, nil
		}
		return nil, err
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
		managed, _ := c.sessionOption(parts[0], "@tmc_managed")
		sessions = append(sessions, SessionSummary{
			Name:     parts[0],
			Windows:  windows,
			Attached: attached,
			Managed:  managed == "1",
		})
	}
	return sessions, nil
}

func (c *Client) SessionExists(session string) (bool, error) {
	err := run([]string{"tmux", "has-session", "-t", session})
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "can't find session") || strings.Contains(err.Error(), "failed to connect") {
		return false, nil
	}
	return false, err
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

func (c *Client) sessionOption(session, option string) (string, error) {
	return runOutput([]string{"tmux", "show-options", "-v", "-t", session, option})
}

func (c *Client) paneOption(paneID, option string) (string, error) {
	return runOutput([]string{"tmux", "show-options", "-p", "-v", "-t", paneID, option})
}

func (c *Client) ensureSessionMissing(session string) error {
	exists, err := c.SessionExists(session)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("session %q already exists", session)
	}
	return nil
}

func run(args []string) error {
	cmd := exec.Command(args[0], args[1:]...)
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
	cmd := exec.Command(args[0], args[1:]...)
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
