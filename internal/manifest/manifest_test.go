package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileResolvesPathsAndHooks(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	manifestPath := filepath.Join(root, "project.yml")
	if err := os.MkdirAll(filepath.Join(root, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}

	body := `version: 1
name: "example"
root: "."
commands:
  server:
    run: "go run ./cmd/example"
    cwd: "./service"
startup_hooks:
  - command: "echo boot"
    cwd: "."
windows:
  - name: "workspace"
    layout: "dev"
    root: "."
    startup_hooks:
      - command: "echo window"
        cwd: "."
    panes:
      - role: "editor"
        cwd: "."
      - role: "logs"
        log_files:
          - "./logs/app.log"
`
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadFile(manifestPath)
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	if m.Root != root {
		t.Fatalf("expected root %q, got %q", root, m.Root)
	}
	if got := m.StartupHooks[0].CWD; got != root {
		t.Fatalf("expected startup hook cwd %q, got %q", root, got)
	}
	if got := m.Commands["server"].CWD; got != filepath.Join(root, "service") {
		t.Fatalf("expected command cwd to resolve, got %q", got)
	}
	if got := m.Windows[0].Panes[1].LogFiles[0]; got != filepath.Join(root, "logs", "app.log") {
		t.Fatalf("expected log file to resolve, got %q", got)
	}
}

func TestRenderInitTemplateProducesQuotedRoot(t *testing.T) {
	body, err := RenderInitTemplate(InitConfig{
		Name:   "Example App",
		Root:   filepath.Join(string(os.PathSeparator), "tmp", "Example App"),
		Layout: "dev",
	})
	if err != nil {
		t.Fatalf("RenderInitTemplate returned error: %v", err)
	}

	if want := `root: "/tmp/Example App"`; !containsLine(body, want) {
		t.Fatalf("expected %q in template:\n%s", want, body)
	}
}

func TestRenderInitTemplateFrontendUsesFrontendCommands(t *testing.T) {
	body, err := RenderInitTemplate(InitConfig{
		Name:   "Example Frontend",
		Root:   filepath.Join(string(os.PathSeparator), "tmp", "Example Frontend"),
		Layout: "frontend",
	})
	if err != nil {
		t.Fatalf("RenderInitTemplate returned error: %v", err)
	}

	for _, want := range []string{
		`layout: frontend`,
		`run: npm run dev`,
		`run: npm test`,
		`- role: server`,
		`- role: tests`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in template:\n%s", want, body)
		}
	}
}

func TestRenderInitTemplateRejectsUnknownLayout(t *testing.T) {
	if _, err := RenderInitTemplate(InitConfig{Root: ".", Layout: "weird"}); err == nil {
		t.Fatal("expected unknown layout error")
	}
}

func containsLine(body, line string) bool {
	for _, candidate := range splitLines(body) {
		if candidate == line {
			return true
		}
	}
	return false
}

func splitLines(body string) []string {
	var lines []string
	start := 0
	for i, r := range body {
		if r == '\n' {
			lines = append(lines, body[start:i])
			start = i + 1
		}
	}
	if start <= len(body) {
		lines = append(lines, body[start:])
	}
	return lines
}
