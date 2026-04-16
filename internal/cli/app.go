package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/justynclarknetwork/tmux-mission-control/internal/doctor"
	"github.com/justynclarknetwork/tmux-mission-control/internal/manifest"
	"github.com/justynclarknetwork/tmux-mission-control/internal/runtime"
	"github.com/justynclarknetwork/tmux-mission-control/internal/status"
	"github.com/justynclarknetwork/tmux-mission-control/internal/tmux"
)

const usageText = `tmc: terminal-first tmux workspace launcher

Usage:
  tmc init [--output project.yml] [--name NAME] [--root PATH] [--layout LAYOUT]
  tmc start --file project.yml [--detach]
  tmc stop --session NAME
  tmc list
  tmc status --session NAME
  tmc doctor [--file project.yml]
  tmc dry-run --file project.yml [--detach]
  tmc completion [bash|zsh|fish]

Layouts:
  dev, backend, frontend, ops, agent-lab
`

func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		_, err := fmt.Fprint(stdout, usageText)
		return err
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], stdout)
	case "start":
		return runStart(args[1:], stdout)
	case "stop":
		return runStop(args[1:], stdout)
	case "list":
		return runList(args[1:], stdout)
	case "status":
		return runStatus(args[1:], stdout)
	case "doctor":
		return runDoctor(args[1:], stdout)
	case "dry-run":
		return runDryRun(args[1:], stdout)
	case "completion":
		return runCompletion(args[1:], stdout)
	case "help", "-h", "--help":
		_, err := fmt.Fprint(stdout, usageText)
		return err
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runInit(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	output := fs.String("output", "project.yml", "manifest path to write")
	name := fs.String("name", "", "project name")
	root := fs.String("root", ".", "project root")
	layout := fs.String("layout", "dev", "default layout")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg := manifest.InitConfig{
		Name:   *name,
		Root:   *root,
		Output: *output,
		Layout: *layout,
	}

	body, err := manifest.RenderInitTemplate(cfg)
	if err != nil {
		return err
	}

	target, err := filepath.Abs(*output)
	if err != nil {
		return err
	}
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "wrote %s\n", target)
	return err
}

func runStart(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	file := fs.String("file", "", "manifest path")
	detach := fs.Bool("detach", false, "create session without attaching")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return errors.New("start requires --file")
	}

	m, err := manifest.LoadFile(*file)
	if err != nil {
		return err
	}

	planner := runtime.NewPlanner()
	plan, err := planner.Plan(m, runtime.StartOptions{Detach: *detach})
	if err != nil {
		return err
	}

	client := tmux.NewClient()
	if err := client.Apply(plan); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "started session %s\n", plan.SessionName)
	return err
}

func runDryRun(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("dry-run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	file := fs.String("file", "", "manifest path")
	detach := fs.Bool("detach", false, "create session without attaching")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return errors.New("dry-run requires --file")
	}

	m, err := manifest.LoadFile(*file)
	if err != nil {
		return err
	}

	planner := runtime.NewPlanner()
	plan, err := planner.Plan(m, runtime.StartOptions{Detach: *detach})
	if err != nil {
		return err
	}

	for _, line := range plan.Describe() {
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	return nil
}

func runStop(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	session := fs.String("session", "", "session name")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *session == "" {
		return errors.New("stop requires --session")
	}

	client := tmux.NewClient()
	if err := client.StopSession(*session); err != nil {
		return err
	}

	_, err := fmt.Fprintf(stdout, "stopped session %s\n", *session)
	return err
}

func runList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return err
	}

	client := tmux.NewClient()
	sessions, err := client.ListSessions()
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		_, err := fmt.Fprintln(stdout, "no tmux sessions")
		return err
	}

	if _, err := fmt.Fprintln(stdout, "SESSION\tWINDOWS\tATTACHED\tMANAGED"); err != nil {
		return err
	}
	for _, session := range sessions {
		managed := "no"
		if session.Managed {
			managed = "yes"
		}
		if _, err := fmt.Fprintf(stdout, "%s\t%d\t%d\t%s\n", session.Name, session.Windows, session.Attached, managed); err != nil {
			return err
		}
	}
	return nil
}

func runStatus(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	session := fs.String("session", "", "session name")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *session == "" {
		return errors.New("status requires --session")
	}

	client := tmux.NewClient()
	report, err := status.Collect(client, *session)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdout, "session: %s\nexists: %t\nwindows: %d\n", report.Session, report.Exists, report.WindowCount); err != nil {
		return err
	}
	if len(report.Panes) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(stdout, "panes:"); err != nil {
		return err
	}
	for _, pane := range report.Panes {
		exitValue := "-"
		if pane.LastExit != nil {
			exitValue = fmt.Sprintf("%d", *pane.LastExit)
		}
		line := fmt.Sprintf("  %s %s role=%s cwd=%s dead=%t cmd=%s exit=%s",
			pane.Window,
			pane.PaneID,
			pane.Role,
			pane.CWD,
			pane.Dead,
			pane.Command,
			exitValue,
		)
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	return nil
}

func runDoctor(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	file := fs.String("file", "", "manifest path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	var m *manifest.Manifest
	var err error
	if *file != "" {
		m, err = manifest.LoadFile(*file)
		if err != nil {
			return err
		}
	}

	report := doctor.Run(m)
	for _, line := range report.Describe() {
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	if !report.OK() {
		return errors.New("doctor found issues")
	}
	return nil
}

func runCompletion(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return errors.New("completion requires one shell name")
	}
	script, err := completionScript(args[0])
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdout, script)
	return err
}

func completionScript(shell string) (string, error) {
	switch strings.ToLower(shell) {
	case "bash":
		return bashCompletion, nil
	case "zsh":
		return zshCompletion, nil
	case "fish":
		return fishCompletion, nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}
