package status

import (
	"os"
	"strconv"
	"strings"

	"github.com/justynclarknetwork/tmux-mission-control/internal/tmux"
)

type Report struct {
	Session     string
	Exists      bool
	WindowCount int
	Panes       []PaneStatus
}

type PaneStatus struct {
	PaneID   string
	Window   string
	Role     string
	Command  string
	CWD      string
	Dead     bool
	LastExit *int
}

func Collect(client *tmux.Client, session string) (*Report, error) {
	exists, err := client.SessionExists(session)
	if err != nil {
		return nil, err
	}
	report := &Report{Session: session, Exists: exists}
	if !exists {
		return report, nil
	}

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
