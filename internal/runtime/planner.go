package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyn-clark/tmux-mission-control/internal/manifest"
)

type StartOptions struct {
	Detach bool
}

type Planner struct{}

type Plan struct {
	SessionName string   `json:"session"`
	Detach      bool     `json:"detach"`
	Actions     []Action `json:"actions"`
}

type Action struct {
	Kind    string   `json:"kind"`
	Command []string `json:"command"`
	Note    string   `json:"note,omitempty"`
	CWD     string   `json:"cwd,omitempty"`
}

type PanePlan struct {
	Pane      manifest.Pane
	SplitFrom int
	Direction string
	Size      string
}

func NewPlanner() *Planner {
	return &Planner{}
}

func (p *Planner) Plan(m *manifest.Manifest, opts StartOptions) (*Plan, error) {
	if m == nil {
		return nil, fmt.Errorf("manifest is nil")
	}

	attach := true
	if m.Attach != nil {
		attach = *m.Attach
	}
	if opts.Detach {
		attach = false
	}

	plan := &Plan{
		SessionName: m.Session,
		Detach:      !attach,
	}

	stateDir, err := sessionStateDir(m.Session)
	if err != nil {
		return nil, err
	}

	plan.Actions = append(plan.Actions, Action{
		Kind:    "check-session",
		Command: []string{"tmux", "has-session", "-t", m.Session},
		Note:    "verify the target session does not already exist",
	})

	for _, hook := range m.StartupHooks {
		plan.Actions = append(plan.Actions, shellAction("startup-hook", hook.Command, firstNonEmpty(hook.CWD, m.Root), mergeEnv(m.Env, hook.Env), hookLabel(hook, "project startup")))
	}

	for windowIndex, window := range m.Windows {
		panes, err := expandWindow(window)
		if err != nil {
			return nil, err
		}

		for _, hook := range window.StartupHooks {
			plan.Actions = append(plan.Actions, shellAction("window-startup-hook", hook.Command, firstNonEmpty(hook.CWD, window.Root), mergeEnv(m.Env, window.Env, hook.Env), hookLabel(hook, fmt.Sprintf("window %s startup", window.Name))))
		}

		for paneIndex, pane := range panes {
			paneCommand, paneCWD, paneEnv, err := resolvePaneCommand(*m, window, pane.Pane)
			if err != nil {
				return nil, err
			}

			targetWindow := fmt.Sprintf("%s:%s", m.Session, window.Name)
			targetPane := fmt.Sprintf("%s.%d", targetWindow, paneIndex)
			stateFile := filepath.Join(stateDir, fmt.Sprintf("%s__%d__%d.exit", sanitize(window.Name), windowIndex, paneIndex))

			if windowIndex == 0 && paneIndex == 0 {
				plan.Actions = append(plan.Actions,
					Action{
						Kind:    "new-session",
						Command: []string{"tmux", "new-session", "-d", "-s", m.Session, "-n", window.Name, "-c", paneCWD, paneShellCommand(m.Shell)},
						Note:    fmt.Sprintf("create session %s and window %s", m.Session, window.Name),
						CWD:     paneCWD,
					},
					setOptionAction(m.Session, "base-index", "0", "normalize window indexing for the session"),
					setOptionAction(m.Session, "pane-base-index", "0", "normalize pane indexing for the session"),
					setOptionAction(m.Session, "default-shell", m.Shell, "record the preferred shell for the session"),
					setOptionAction(m.Session, "@tmc_managed", "1", "mark session as managed"),
					setOptionAction(m.Session, "@tmc_project", m.Name, "record project name"),
					setOptionAction(m.Session, "@tmc_manifest", m.ManifestPath, "record manifest path"),
				)
			} else if paneIndex == 0 {
				plan.Actions = append(plan.Actions, Action{
					Kind:    "new-window",
					Command: []string{"tmux", "new-window", "-d", "-t", m.Session, "-n", window.Name, "-c", paneCWD, paneShellCommand(m.Shell)},
					Note:    fmt.Sprintf("create window %s", window.Name),
					CWD:     paneCWD,
				})
			} else {
				args := []string{"tmux", "split-window", "-d", "-t", fmt.Sprintf("%s.%d", targetWindow, pane.SplitFrom)}
				if pane.Direction == "h" {
					args = append(args, "-h")
				} else {
					args = append(args, "-v")
				}
				if pane.Size != "" {
					args = append(args, "-p", pane.Size)
				}
				args = append(args, "-c", paneCWD, paneShellCommand(m.Shell))
				plan.Actions = append(plan.Actions, Action{
					Kind:    "split-window",
					Command: args,
					Note:    fmt.Sprintf("create %s pane in window %s", pane.Pane.Role, window.Name),
					CWD:     paneCWD,
				})
			}

			plan.Actions = append(plan.Actions,
				setPaneOptionAction(targetPane, "@tmc_role", pane.Pane.Role, "record pane role"),
				setPaneOptionAction(targetPane, "@tmc_command", paneCommand, "record pane command"),
				setPaneOptionAction(targetPane, "@tmc_cwd", paneCWD, "record pane cwd"),
				setPaneOptionAction(targetPane, "@tmc_state_file", stateFile, "record pane state file"),
			)

			if pane.Pane.Title != "" {
				plan.Actions = append(plan.Actions, Action{
					Kind:    "select-pane-title",
					Command: []string{"tmux", "select-pane", "-t", targetPane, "-T", pane.Pane.Title},
					Note:    fmt.Sprintf("set title for %s", targetPane),
				})
			}

			setupCommand := buildPaneSetupScript(paneCWD, paneEnv, paneCommand, stateFile)
			if strings.TrimSpace(setupCommand) != "" {
				plan.Actions = append(plan.Actions, Action{
					Kind:    "send-keys",
					Command: []string{"tmux", "send-keys", "-t", targetPane, setupCommand, "C-m"},
					Note:    fmt.Sprintf("dispatch command for %s", targetPane),
					CWD:     paneCWD,
				})
			}
		}
	}

	if attach {
		if os.Getenv("TMUX") != "" {
			plan.Actions = append(plan.Actions, Action{
				Kind:    "attach",
				Command: []string{"tmux", "switch-client", "-t", m.Session},
				Note:    "switch current tmux client to the managed session",
			})
		} else {
			plan.Actions = append(plan.Actions, Action{
				Kind:    "attach",
				Command: []string{"tmux", "attach-session", "-t", m.Session},
				Note:    "attach terminal to the managed session",
			})
		}
	}

	return plan, nil
}

func (p *Plan) Describe() []string {
	lines := []string{
		fmt.Sprintf("session: %s", p.SessionName),
		fmt.Sprintf("detach: %t", p.Detach),
		"actions:",
	}
	for i, action := range p.Actions {
		lines = append(lines, fmt.Sprintf("%02d. %s", i+1, formatCommand(action.Command)))
		if action.Note != "" {
			lines = append(lines, fmt.Sprintf("    %s", action.Note))
		}
	}
	return lines
}

func formatCommand(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return strings.Join(quoted, " ")
}

func ResolvePaneCommand(m manifest.Manifest, window manifest.Window, pane manifest.Pane) (string, string, map[string]string, error) {
	cwd := firstNonEmpty(pane.CWD, window.Root, m.Root)
	env := mergeEnv(m.Env, window.Env, pane.Env)
	command := strings.TrimSpace(pane.Command)

	if pane.CommandRef != "" {
		def := m.Commands[pane.CommandRef]
		command = def.Run
		if def.CWD != "" {
			cwd = def.CWD
		}
		env = mergeEnv(env, def.Env)
	}

	if command == "" {
		switch pane.Role {
		case "editor":
			command = "${EDITOR:-nvim} ."
		case "logs":
			if len(pane.LogFiles) > 0 {
				parts := make([]string, 0, len(pane.LogFiles)+2)
				parts = append(parts, "tail", "-F")
				parts = append(parts, pane.LogFiles...)
				command = shellJoin(parts)
			}
		}
	}

	return command, cwd, env, nil
}

func resolvePaneCommand(m manifest.Manifest, window manifest.Window, pane manifest.Pane) (string, string, map[string]string, error) {
	return ResolvePaneCommand(m, window, pane)
}

func shellAction(kind, command, cwd string, env map[string]string, note string) Action {
	return Action{
		Kind:    kind,
		Command: []string{"/bin/sh", "-lc", buildShellCommand(cwd, env, command)},
		Note:    note,
		CWD:     cwd,
	}
}

func hookLabel(hook manifest.Hook, fallback string) string {
	if hook.Name != "" {
		return hook.Name
	}
	return fallback
}

func setOptionAction(target, name, value, note string) Action {
	return Action{
		Kind:    "set-option",
		Command: []string{"tmux", "set-option", "-t", target, "-q", name, value},
		Note:    note,
	}
}

func setPaneOptionAction(target, name, value, note string) Action {
	return Action{
		Kind:    "set-option",
		Command: []string{"tmux", "set-option", "-p", "-t", target, "-q", name, value},
		Note:    note,
	}
}

func buildPaneSetupScript(cwd string, env map[string]string, command string, stateFile string) string {
	parts := []string{
		fmt.Sprintf("mkdir -p %s", shellQuote(filepath.Dir(stateFile))),
		fmt.Sprintf("cd %s", shellQuote(cwd)),
	}
	keys := sortedKeys(env)
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("export %s=%s", key, shellQuote(env[key])))
	}
	if strings.TrimSpace(command) != "" {
		parts = append(parts,
			fmt.Sprintf("{ %s; }; rc=$?; printf '%%s\\n' \"$rc\" > %s", command, shellQuote(stateFile)),
		)
	}
	return strings.Join(parts, "; ")
}

func buildShellCommand(cwd string, env map[string]string, command string) string {
	parts := []string{
		fmt.Sprintf("cd %s", shellQuote(cwd)),
	}
	keys := sortedKeys(env)
	for _, key := range keys {
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

func shellJoin(parts []string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		quoted = append(quoted, shellQuote(part))
	}
	return strings.Join(quoted, " ")
}

func sessionStateDir(session string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "tmc", sanitize(session)), nil
}

func sanitize(value string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(value)
}

func paneShellCommand(shell string) string {
	return fmt.Sprintf("exec %s -i", shellQuote(shell))
}
