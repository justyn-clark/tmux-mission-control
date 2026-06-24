package status

import (
	"os"
	"strconv"
	"strings"

	"github.com/justynclarknetwork/tmux-mission-control/internal/tmux"
)

type Report struct {
	Session      string       `json:"session"`
	TmuxRunning  bool         `json:"tmux_running"`
	Exists       bool         `json:"exists"`
	Managed      bool         `json:"managed"`
	Project      string       `json:"project,omitempty"`
	ManifestPath string       `json:"manifest_path,omitempty"`
	WindowCount  int          `json:"windows"`
	Panes        []PaneStatus `json:"panes,omitempty"`
}

type PaneStatus struct {
	PaneID   string `json:"pane_id"`
	Window   string `json:"window"`
	Role     string `json:"role"`
	Command  string `json:"command"`
	CWD      string `json:"cwd"`
	Dead     bool   `json:"dead"`
	LastExit *int   `json:"last_exit,omitempty"`
}

func Collect(client *tmux.Client, session string) (*Report, error) {
	exists, err := client.SessionExists(session)
	if err != nil {
		if tmux.IsTmuxNotRunning(err) {
			return &Report{Session: session, TmuxRunning: false, Exists: false}, nil
		}
		return nil, err
	}
	report := &Report{Session: session, TmuxRunning: true, Exists: exists}
	if !exists {
		return report, nil
	}

	metadata, err := client.SessionMetadata(session)
	if err != nil {
		return nil, err
	}
	report.Managed = metadata.Managed
	report.Project = metadata.Project
	report.ManifestPath = metadata.ManifestPath

	windowCount, err := client.WindowCount(session)
	if err != nil {
		return nil, err
	}
	report.WindowCount = windowCount

	panes, err := client.ListPanes(session)
	if err != nil {
		return nil, err
	}

	for _, pane := range panes {
		report.Panes = append(report.Panes, PaneStatus{
			PaneID:   pane.PaneID,
			Window:   pane.Window,
			Role:     pane.Role,
			Command:  pane.Command,
			CWD:      pane.CWD,
			Dead:     pane.Dead,
			LastExit: readLastExit(pane.StateFile),
		})
	}
	return report, nil
}

func readLastExit(path string) *int {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(body)))
	if err != nil {
		return nil
	}
	return &value
}
