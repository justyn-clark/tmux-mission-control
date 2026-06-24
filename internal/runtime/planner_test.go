package runtime

import (
	"path/filepath"
	"testing"

	"github.com/justyn-clark/tmux-mission-control/internal/manifest"
)

func TestPlanIncludesSessionBootstrapAndNoShutdownHooks(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, ".cache"))
	m := &manifest.Manifest{
		Version: 1,
		Name:    "example",
		Root:    root,
		Session: "example",
		Shell:   "/bin/sh",
		Commands: map[string]manifest.CommandDef{
			"server": {Run: "go run ./cmd/example"},
		},
		ShutdownHooks: []manifest.Hook{
			{Command: "echo bye"},
		},
		Windows: []manifest.Window{
			{
				Name:   "workspace",
				Layout: "backend",
				Root:   root,
				Panes: []manifest.Pane{
					{Role: "editor"},
					{Role: "shell"},
					{Role: "server", CommandRef: "server"},
				},
			},
		},
	}

	plan, err := NewPlanner().Plan(m, StartOptions{})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}

	var sawBootstrap bool
	var sawStateFile bool
	for _, action := range plan.Actions {
		if action.Kind == "new-session" {
			sawBootstrap = true
		}
		for _, arg := range action.Command {
			if arg == "@tmc_state_file" {
				sawStateFile = true
			}
			if arg == "echo bye" {
				t.Fatalf("shutdown hook leaked into start plan")
			}
		}
	}

	if !sawBootstrap {
		t.Fatalf("expected new-session action")
	}
	if !sawStateFile {
		t.Fatalf("expected pane state file metadata action")
	}

	expectedPrefix, err := sessionStateDir("example")
	if err != nil {
		t.Fatalf("sessionStateDir returned error: %v", err)
	}
	found := false
	for _, action := range plan.Actions {
		for _, arg := range action.Command {
			if len(arg) >= len(expectedPrefix) && arg[:len(expectedPrefix)] == expectedPrefix {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected state path prefix %q in plan actions", expectedPrefix)
	}
}
