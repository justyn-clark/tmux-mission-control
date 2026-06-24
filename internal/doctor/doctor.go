package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/justyn-clark/tmux-mission-control/internal/manifest"
	"github.com/justyn-clark/tmux-mission-control/internal/runtime"
)

type Report struct {
	Checks []Check
}

type Check struct {
	Name   string
	Status string
	Detail string
}

func Run(m *manifest.Manifest) Report {
	report := Report{}

	report.add(checkBinary("tmux", "tmux installed"))
	report.add(checkBinary(defaultShell(m), "shell available"))
	report.add(checkEditor(m))

	if m == nil {
		return report
	}

	report.add(checkPath("project root", m.Root, true))
	report.add(checkCommands(m))
	report.add(checkHooks("startup hooks", m.Root, m.StartupHooks))
	report.add(checkHooks("shutdown hooks", m.Root, m.ShutdownHooks))

	for _, window := range m.Windows {
		report.add(checkPath(fmt.Sprintf("window %s root", window.Name), window.Root, true))
		report.add(checkHooks(fmt.Sprintf("window %s startup hooks", window.Name), window.Root, window.StartupHooks))
		report.add(checkHooks(fmt.Sprintf("window %s shutdown hooks", window.Name), window.Root, window.ShutdownHooks))
		for _, pane := range window.Panes {
			cwd := pane.CWD
			if cwd == "" {
				cwd = window.Root
			}
			report.add(checkPath(fmt.Sprintf("pane %s cwd", pane.Role), cwd, true))
			if len(pane.LogFiles) > 0 {
				for _, file := range pane.LogFiles {
					report.add(checkPath(fmt.Sprintf("log file %s", filepath.Base(file)), file, false))
				}
			}
			command, _, _, err := runtime.ResolvePaneCommand(*m, window, pane)
			if err != nil {
				report.add(Check{Name: fmt.Sprintf("pane %s command", pane.Role), Status: "fail", Detail: err.Error()})
				continue
			}
			if strings.TrimSpace(command) != "" {
				report.add(checkCommandLine(fmt.Sprintf("pane %s command", pane.Role), command, defaultShell(m)))
			}
		}
	}

	return report
}

func (r Report) Describe() []string {
	lines := make([]string, 0, len(r.Checks))
	for _, check := range r.Checks {
		lines = append(lines, fmt.Sprintf("[%s] %s: %s", strings.ToUpper(check.Status), check.Name, check.Detail))
	}
	return lines
}

func (r Report) OK() bool {
	for _, check := range r.Checks {
		if check.Status == "fail" {
			return false
		}
	}
	return true
}

func (r *Report) add(check Check) {
	r.Checks = append(r.Checks, check)
}

func checkBinary(binary, name string) Check {
	if binary == "" {
		return Check{Name: name, Status: "fail", Detail: "binary name is empty"}
	}
	if _, err := exec.LookPath(binary); err != nil {
		return Check{Name: name, Status: "fail", Detail: err.Error()}
	}
	return Check{Name: name, Status: "pass", Detail: binary}
}

func checkEditor(m *manifest.Manifest) Check {
	editor := "nvim"
	if envEditor := os.Getenv("EDITOR"); envEditor != "" {
		editor = strings.Fields(envEditor)[0]
	}
	if m != nil {
		for _, window := range m.Windows {
			for _, pane := range window.Panes {
				if pane.Role == "editor" && pane.Command != "" {
					head := commandHead(pane.Command)
					if head != "" {
						editor = head
					}
				}
			}
		}
	}
	return checkBinary(editor, "editor exists")
}

func checkPath(name, path string, directory bool) Check {
	info, err := os.Stat(path)
	if err != nil {
		return Check{Name: name, Status: "fail", Detail: err.Error()}
	}
	if directory && !info.IsDir() {
		return Check{Name: name, Status: "fail", Detail: "path is not a directory"}
	}
	if !directory && info.IsDir() {
		return Check{Name: name, Status: "fail", Detail: "path is a directory"}
	}
	return Check{Name: name, Status: "pass", Detail: path}
}

func checkCommands(m *manifest.Manifest) Check {
	for name, def := range m.Commands {
		if result := checkCommandLine(fmt.Sprintf("command %s", name), def.Run, defaultShell(m)); result.Status == "fail" {
			return result
		}
	}
	return Check{Name: "manifest commands", Status: "pass", Detail: fmt.Sprintf("%d command definitions", len(m.Commands))}
}

func checkHooks(name, base string, hooks []manifest.Hook) Check {
	for _, hook := range hooks {
		if result := checkCommandLine(name, hook.Command, "/bin/sh"); result.Status == "fail" {
			return result
		}
		if hook.CWD != "" {
			if result := checkPath(name+" cwd", hook.CWD, true); result.Status == "fail" {
				return result
			}
		} else if base != "" {
			if result := checkPath(name+" cwd", base, true); result.Status == "fail" {
				return result
			}
		}
	}
	return Check{Name: name, Status: "pass", Detail: fmt.Sprintf("%d hooks", len(hooks))}
}

func checkCommandLine(name, command, shell string) Check {
	head := commandHead(command)
	if head == "" {
		return Check{Name: name, Status: "pass", Detail: "empty command"}
	}
	cmd := exec.Command(shell, "-lc", "command -v "+shellEscape(head)+" >/dev/null")
	if err := cmd.Run(); err != nil {
		return Check{Name: name, Status: "fail", Detail: fmt.Sprintf("unresolved executable %q in %q", head, command)}
	}
	return Check{Name: name, Status: "pass", Detail: command}
}

func defaultShell(m *manifest.Manifest) string {
	if m != nil && m.Shell != "" {
		return m.Shell
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}

func commandHead(command string) string {
	fields := splitShellWords(command)
	for _, field := range fields {
		if strings.Contains(field, "=") && !strings.HasPrefix(field, "./") && !strings.HasPrefix(field, "/") {
			continue
		}
		if expanded := shellExpansionHead(field); expanded != "" {
			return expanded
		}
		return field
	}
	return ""
}

func shellExpansionHead(field string) string {
	if strings.HasPrefix(field, "${") && strings.HasSuffix(field, "}") {
		inner := strings.TrimSuffix(strings.TrimPrefix(field, "${"), "}")
		if idx := strings.Index(inner, ":-"); idx >= 0 && idx+2 < len(inner) {
			return inner[idx+2:]
		}
		if value := os.Getenv(inner); value != "" {
			return splitShellWords(value)[0]
		}
	}
	if strings.HasPrefix(field, "$") && len(field) > 1 {
		if value := os.Getenv(strings.TrimPrefix(field, "$")); value != "" {
			return splitShellWords(value)[0]
		}
	}
	return ""
}

func splitShellWords(input string) []string {
	var fields []string
	var current strings.Builder
	var quote rune
	escaped := false
	for _, r := range input {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	return fields
}

func shellEscape(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
