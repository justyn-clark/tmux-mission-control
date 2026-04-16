package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

var validRoles = []string{"editor", "shell", "tests", "logs", "server", "agent", "docs"}

type Manifest struct {
	Version       int                   `yaml:"version"`
	Name          string                `yaml:"name"`
	Root          string                `yaml:"root"`
	Session       string                `yaml:"session"`
	Shell         string                `yaml:"shell"`
	Attach        *bool                 `yaml:"attach"`
	Env           map[string]string     `yaml:"env"`
	Commands      map[string]CommandDef `yaml:"commands"`
	StartupHooks  []Hook                `yaml:"startup_hooks"`
	ShutdownHooks []Hook                `yaml:"shutdown_hooks"`
	Windows       []Window              `yaml:"windows"`
	ManifestPath  string                `yaml:"-"`
}

type CommandDef struct {
	Run         string            `yaml:"run"`
	CWD         string            `yaml:"cwd"`
	Env         map[string]string `yaml:"env"`
	Description string            `yaml:"description"`
}

type Hook struct {
	Name    string            `yaml:"name"`
	Command string            `yaml:"command"`
	CWD     string            `yaml:"cwd"`
	Env     map[string]string `yaml:"env"`
}

type Window struct {
	Name          string            `yaml:"name"`
	Layout        string            `yaml:"layout"`
	Root          string            `yaml:"root"`
	Env           map[string]string `yaml:"env"`
	StartupHooks  []Hook            `yaml:"startup_hooks"`
	ShutdownHooks []Hook            `yaml:"shutdown_hooks"`
	Panes         []Pane            `yaml:"panes"`
}

type Pane struct {
	Title      string            `yaml:"title"`
	Role       string            `yaml:"role"`
	CWD        string            `yaml:"cwd"`
	Command    string            `yaml:"command"`
	CommandRef string            `yaml:"command_ref"`
	Env        map[string]string `yaml:"env"`
	LogFiles   []string          `yaml:"log_files"`
}

type InitConfig struct {
	Name   string
	Root   string
	Output string
	Layout string
}

func (c *CommandDef) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		c.Run = node.Value
		return nil
	case yaml.MappingNode:
		type alias CommandDef
		var value alias
		if err := node.Decode(&value); err != nil {
			return err
		}
		*c = CommandDef(value)
		return nil
	default:
		return fmt.Errorf("invalid command definition at line %d", node.Line)
	}
}

func (h *Hook) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		h.Command = node.Value
		return nil
	case yaml.MappingNode:
		type alias Hook
		var value alias
		if err := node.Decode(&value); err != nil {
			return err
		}
		*h = Hook(value)
		return nil
	default:
		return fmt.Errorf("invalid hook definition at line %d", node.Line)
	}
}

func LoadFile(path string) (*Manifest, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	var m Manifest
	decoder := yaml.NewDecoder(bytes.NewReader(body))
	decoder.KnownFields(true)
	if err := decoder.Decode(&m); err != nil {
		return nil, err
	}

	m.ManifestPath = absPath
	if err := m.Normalize(); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Manifest) Normalize() error {
	if m.Version == 0 {
		m.Version = 1
	}
	if m.Name == "" {
		return errors.New("manifest name is required")
	}
	if m.Root == "" {
		return errors.New("manifest root is required")
	}
	if m.Session == "" {
		m.Session = sanitizeName(m.Name)
	}
	if m.Shell == "" {
		if shell := os.Getenv("SHELL"); shell != "" {
			m.Shell = shell
		} else {
			m.Shell = "/bin/sh"
		}
	}
	root, err := resolvePath(baseDir(m.ManifestPath), m.Root)
	if err != nil {
		return err
	}
	m.Root = root
	if m.Env == nil {
		m.Env = map[string]string{}
	}
	if len(m.Windows) == 0 {
		return errors.New("manifest requires at least one window")
	}

	windowNames := map[string]struct{}{}
	for i := range m.Windows {
		w := &m.Windows[i]
		if w.Name == "" {
			return fmt.Errorf("window %d is missing a name", i)
		}
		if _, exists := windowNames[w.Name]; exists {
			return fmt.Errorf("duplicate window name %q", w.Name)
		}
		windowNames[w.Name] = struct{}{}
		if w.Layout == "" {
			w.Layout = "dev"
		}
		if w.Root == "" {
			w.Root = m.Root
		} else {
			w.Root, err = resolvePath(m.Root, w.Root)
			if err != nil {
				return err
			}
		}
		if w.Env == nil {
			w.Env = map[string]string{}
		}
		for j := range w.Panes {
			p := &w.Panes[j]
			if p.Role != "" && !slices.Contains(validRoles, p.Role) {
				return fmt.Errorf("window %q pane %d has invalid role %q", w.Name, j, p.Role)
			}
			if p.CWD != "" {
				p.CWD, err = resolvePath(w.Root, p.CWD)
				if err != nil {
					return err
				}
			}
			for k := range p.LogFiles {
				p.LogFiles[k], err = resolvePath(w.Root, p.LogFiles[k])
				if err != nil {
					return err
				}
			}
		}
		if len(w.Panes) == 0 {
			return fmt.Errorf("window %q requires at least one pane or a layout expansion", w.Name)
		}
	}

	for name, command := range m.Commands {
		if command.Run == "" {
			return fmt.Errorf("command %q requires run", name)
		}
		if command.CWD != "" {
			command.CWD, err = resolvePath(m.Root, command.CWD)
			if err != nil {
				return err
			}
			m.Commands[name] = command
		}
	}

	if err := validateHooks(m.Root, &m.StartupHooks); err != nil {
		return err
	}
	if err := validateHooks(m.Root, &m.ShutdownHooks); err != nil {
		return err
	}
	for i := range m.Windows {
		w := &m.Windows[i]
		if err := validateHooks(w.Root, &w.StartupHooks); err != nil {
			return fmt.Errorf("window %q: %w", w.Name, err)
		}
		if err := validateHooks(w.Root, &w.ShutdownHooks); err != nil {
			return fmt.Errorf("window %q: %w", w.Name, err)
		}
		for _, p := range w.Panes {
			if p.Command != "" && p.CommandRef != "" {
				return fmt.Errorf("pane %q in window %q cannot set both command and command_ref", p.Role, w.Name)
			}
			if p.CommandRef != "" {
				if _, ok := m.Commands[p.CommandRef]; !ok {
					return fmt.Errorf("pane %q in window %q references unknown command %q", p.Role, w.Name, p.CommandRef)
				}
			}
		}
	}

	return nil
}

func validateHooks(base string, hooks *[]Hook) error {
	for i := range *hooks {
		if (*hooks)[i].Command == "" {
			return fmt.Errorf("hook %d requires command", i)
		}
		if (*hooks)[i].CWD != "" {
			cwd, err := resolvePath(base, (*hooks)[i].CWD)
			if err != nil {
				return err
			}
			(*hooks)[i].CWD = cwd
		}
	}
	return nil
}

func sanitizeName(name string) string {
	lower := strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9_-]+`)
	clean := re.ReplaceAllString(lower, "-")
	clean = strings.Trim(clean, "-")
	if clean == "" {
		return "tmc"
	}
	return clean
}

func baseDir(path string) string {
	if path == "" {
		dir, _ := os.Getwd()
		return dir
	}
	return filepath.Dir(path)
}

func resolvePath(base, path string) (string, error) {
	if path == "" {
		return base, nil
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	return filepath.Abs(filepath.Join(base, path))
}

func RenderInitTemplate(cfg InitConfig) (string, error) {
	root, err := filepath.Abs(cfg.Root)
	if err != nil {
		return "", err
	}
	name := cfg.Name
	if name == "" {
		name = filepath.Base(root)
	}
	layout := cfg.Layout
	if layout == "" {
		layout = "dev"
	}
	if !isSupportedLayout(layout) {
		return "", fmt.Errorf("unsupported init layout %q", layout)
	}
	session := sanitizeName(name)

	blueprint := initBlueprintForLayout(layout)

	return fmt.Sprintf(`version: 1
name: %q
root: %q
session: %q
shell: %q
attach: true
env:
  TMC_PROJECT: %q
commands:
%s
windows:
  - name: workspace
    layout: %s
    panes:
%s
`, name, root, session, shellOrDefault(), name, blueprint.commandsYAML, layout, blueprint.panesYAML), nil
}

type initBlueprint struct {
	commandsYAML string
	panesYAML    string
}

func isSupportedLayout(layout string) bool {
	_, ok := map[string]struct{}{
		"dev":       {},
		"backend":   {},
		"frontend":  {},
		"ops":       {},
		"agent-lab": {},
	}[layout]
	return ok
}

func initBlueprintForLayout(layout string) initBlueprint {
	switch layout {
	case "backend":
		return initBlueprint{
			commandsYAML: `  server:
    run: go run ./...
  logs:
    run: tail -F ./tmp/app.log`,
			panesYAML: `      - role: editor
        command: "${EDITOR:-nvim} ."
      - role: shell
      - role: server
        command_ref: server
      - role: logs
        command_ref: logs`,
		}
	case "frontend":
		return initBlueprint{
			commandsYAML: `  server:
    run: npm run dev
  tests:
    run: npm test
  logs:
    run: tail -F ./tmp/frontend.log`,
			panesYAML: `      - role: editor
        command: "${EDITOR:-nvim} ."
      - role: shell
      - role: server
        command_ref: server
      - role: tests
        command_ref: tests
      - role: logs
        command_ref: logs`,
		}
	case "ops":
		return initBlueprint{
			commandsYAML: `  service:
    run: make run
  logs:
    run: tail -F ./tmp/service.log
  docs:
    run: less README.md`,
			panesYAML: `      - role: shell
      - role: server
        command_ref: service
      - role: logs
        command_ref: logs
      - role: docs
        command_ref: docs`,
		}
	case "agent-lab":
		return initBlueprint{
			commandsYAML: `  tests:
    run: npm test
  logs:
    run: tail -F ./tmp/agent.log
  agent:
    run: codex
  docs:
    run: less README.md`,
			panesYAML: `      - role: editor
        command: "${EDITOR:-nvim} ."
      - role: shell
      - role: tests
        command_ref: tests
      - role: logs
        command_ref: logs
      - role: agent
        command_ref: agent
      - role: docs
        command_ref: docs`,
		}
	default:
		return initBlueprint{
			commandsYAML: `  tests:
    run: go test ./...
  logs:
    run: tail -F ./tmp/app.log`,
			panesYAML: `      - role: editor
        command: "${EDITOR:-nvim} ."
      - role: shell
      - role: tests
        command_ref: tests
      - role: logs
        command_ref: logs`,
		}
	}
}

func shellOrDefault() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/sh"
}
