package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLifecycleWithIsolatedTmuxSocket(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux is not installed")
	}

	tmp, err := os.MkdirTemp("/tmp", "tmc-it-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tmp)
	})
	socket := filepath.Join(tmp, "tmux.sock")
	cacheDir := filepath.Join(tmp, "cache")
	root := filepath.Join(tmp, "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TMC_TMUX_SOCKET", socket)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("TMUX", "")
	t.Cleanup(func() {
		_ = exec.Command("tmux", "-S", socket, "kill-server").Run()
	})

	manifestPath := filepath.Join(tmp, "project.yml")
	body := `version: 1
name: "tmc integration"
root: "` + root + `"
session: "tmc-integration"
shell: "/bin/sh"
attach: false
commands:
  done:
    run: "printf done"
shutdown_hooks:
  - command: "printf stopped > shutdown.txt"
windows:
  - name: "workspace"
    layout: "dev"
    panes:
      - role: "shell"
        command_ref: "done"
`
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	var statusBefore map[string]any
	runJSON(t, []string{"status", "--session", "tmc-integration", "--json"}, &statusBefore)
	if statusBefore["tmux_running"].(bool) {
		t.Fatalf("expected isolated tmux server to be stopped before lifecycle, got %#v", statusBefore)
	}

	var dryRun map[string]any
	runJSON(t, []string{"dry-run", "--file", manifestPath, "--json"}, &dryRun)
	if dryRun["session"] != "tmc-integration" {
		t.Fatalf("unexpected dry-run session: %#v", dryRun)
	}

	runCLI(t, []string{"start", "--file", manifestPath, "--detach"})

	var sessions []map[string]any
	runJSON(t, []string{"list", "--managed", "--json"}, &sessions)
	if len(sessions) != 1 || sessions[0]["name"] != "tmc-integration" || sessions[0]["managed"] != true {
		t.Fatalf("unexpected managed sessions: %#v", sessions)
	}

	var statusReport map[string]any
	waitFor(t, time.Second, func() bool {
		statusReport = map[string]any{}
		runJSON(t, []string{"status", "--session", "tmc-integration", "--json"}, &statusReport)
		panes, _ := statusReport["panes"].([]any)
		if len(panes) != 1 {
			return false
		}
		pane, _ := panes[0].(map[string]any)
		return pane["last_exit"] == float64(0)
	})
	if statusReport["project"] != "tmc integration" || statusReport["manifest_path"] != manifestPath {
		t.Fatalf("status did not surface project and manifest metadata: %#v", statusReport)
	}

	runCLI(t, []string{"stop", "--session", "tmc-integration"})
	if _, err := os.Stat(filepath.Join(root, "shutdown.txt")); err != nil {
		t.Fatalf("shutdown hook did not run: %v", err)
	}
}

func TestStartPreflightRejectsMissingPaneCWDWithoutCreatingSession(t *testing.T) {
	env := setupIsolatedTmux(t)
	root := filepath.Join(env.tmp, "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(env.tmp, "missing-cwd.yml")
	body := `version: 1
name: "missing cwd"
root: "` + root + `"
session: "missing-cwd"
shell: "/bin/sh"
attach: false
windows:
  - name: "workspace"
    layout: "dev"
    panes:
      - role: "shell"
        cwd: "./does-not-exist"
`
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := runCLIFail(t, []string{"start", "--file", manifestPath, "--detach"})
	if !strings.Contains(err.Error(), "preflight failed") {
		t.Fatalf("expected preflight failure, got %v", err)
	}

	var statusReport map[string]any
	runJSON(t, []string{"status", "--session", "missing-cwd", "--json"}, &statusReport)
	if statusReport["tmux_running"].(bool) || statusReport["exists"].(bool) {
		t.Fatalf("preflight failure created tmux state: %#v", statusReport)
	}
}

func TestManagedStopFailsWhenRecordedManifestIsMissing(t *testing.T) {
	env := setupIsolatedTmux(t)
	root := filepath.Join(env.tmp, "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(env.tmp, "project.yml")
	body := `version: 1
name: "missing manifest"
root: "` + root + `"
session: "missing-manifest"
shell: "/bin/sh"
attach: false
windows:
  - name: "workspace"
    layout: "dev"
    panes:
      - role: "shell"
`
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	runCLI(t, []string{"start", "--file", manifestPath, "--detach"})
	if err := os.Remove(manifestPath); err != nil {
		t.Fatal(err)
	}

	err := runCLIFail(t, []string{"stop", "--session", "missing-manifest"})
	if !strings.Contains(err.Error(), "recorded unreadable shutdown hook manifest") {
		t.Fatalf("expected missing manifest stop failure, got %v", err)
	}

	var statusReport map[string]any
	runJSON(t, []string{"status", "--session", "missing-manifest", "--json"}, &statusReport)
	if statusReport["exists"] != true {
		t.Fatalf("session should remain after failed audited stop: %#v", statusReport)
	}
}

type isolatedTmuxEnv struct {
	tmp    string
	socket string
}

func setupIsolatedTmux(t *testing.T) isolatedTmuxEnv {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux is not installed")
	}

	tmp, err := os.MkdirTemp("/tmp", "tmc-it-")
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(tmp, "tmux.sock")
	cacheDir := filepath.Join(tmp, "cache")

	t.Setenv("TMC_TMUX_SOCKET", socket)
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("TMUX", "")
	t.Cleanup(func() {
		_ = exec.Command("tmux", "-S", socket, "kill-server").Run()
		_ = os.RemoveAll(tmp)
	})

	return isolatedTmuxEnv{tmp: tmp, socket: socket}
}

func runCLI(t *testing.T, args []string) string {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run(args, &stdout, &stderr); err != nil {
		t.Fatalf("tmc %v failed: %v\nstdout:\n%s\nstderr:\n%s", args, err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func runCLIFail(t *testing.T, args []string) error {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(args, &stdout, &stderr)
	if err == nil {
		t.Fatalf("tmc %v succeeded unexpectedly\nstdout:\n%s\nstderr:\n%s", args, stdout.String(), stderr.String())
	}
	return err
}

func runJSON(t *testing.T, args []string, target any) {
	t.Helper()
	output := runCLI(t, args)
	if err := json.Unmarshal([]byte(output), target); err != nil {
		t.Fatalf("tmc %v returned invalid JSON: %v\n%s", args, err, output)
	}
}

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
