package tmux

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/tmux-mission-control/internal/runtime"
)

func TestApplyCleansUpSessionAfterFailedMutation(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux is not installed")
	}

	tmp, err := os.MkdirTemp("/tmp", "tmc-client-it-")
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(tmp, "tmux.sock")
	root := filepath.Join(tmp, "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMC_TMUX_SOCKET", socket)
	t.Cleanup(func() {
		_ = exec.Command("tmux", "-S", socket, "kill-server").Run()
		_ = os.RemoveAll(tmp)
	})

	client := NewClient()
	err = client.Apply(&runtime.Plan{
		SessionName: "partial-cleanup",
		Actions: []runtime.Action{
			{
				Kind:    "check-session",
				Command: []string{"tmux", "has-session", "-t", "partial-cleanup"},
				Note:    "verify missing session",
			},
			{
				Kind:    "new-session",
				Command: []string{"tmux", "new-session", "-d", "-s", "partial-cleanup", "-n", "workspace", "-c", root, "exec /bin/sh -i"},
				Note:    "create partial-cleanup",
				CWD:     root,
			},
			{
				Kind:    "split-window",
				Command: []string{"tmux", "split-window", "-d", "-t", "partial-cleanup:missing.0", "-c", root, "exec /bin/sh -i"},
				Note:    "force split failure",
				CWD:     root,
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "force split failure") {
		t.Fatalf("expected forced split failure, got %v", err)
	}

	exists, err := client.SessionExists("partial-cleanup")
	if err != nil && !IsTmuxNotRunning(err) {
		t.Fatalf("SessionExists returned unexpected error: %v", err)
	}
	if exists {
		t.Fatal("partial session still exists after failed Apply")
	}
}
