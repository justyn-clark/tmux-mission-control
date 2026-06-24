package runtime

import (
	"fmt"

	"github.com/justyn-clark/tmux-mission-control/internal/manifest"
)

type LayoutPreset struct {
	Name  string
	Slots []PaneSlot
}

type PaneSlot struct {
	DefaultRole  string
	DefaultTitle string
	SplitFrom    int
	Direction    string
	Size         string
}

var presets = map[string]LayoutPreset{
	"dev": {
		Name: "dev",
		Slots: []PaneSlot{
			{DefaultRole: "editor", DefaultTitle: "editor", SplitFrom: -1},
			{DefaultRole: "shell", DefaultTitle: "shell", SplitFrom: 0, Direction: "h", Size: "40"},
			{DefaultRole: "tests", DefaultTitle: "tests", SplitFrom: 1, Direction: "v", Size: "50"},
			{DefaultRole: "logs", DefaultTitle: "logs", SplitFrom: 2, Direction: "v", Size: "50"},
		},
	},
	"backend": {
		Name: "backend",
		Slots: []PaneSlot{
			{DefaultRole: "editor", DefaultTitle: "editor", SplitFrom: -1},
			{DefaultRole: "shell", DefaultTitle: "shell", SplitFrom: 0, Direction: "h", Size: "45"},
			{DefaultRole: "server", DefaultTitle: "server", SplitFrom: 1, Direction: "v", Size: "60"},
			{DefaultRole: "logs", DefaultTitle: "logs", SplitFrom: 2, Direction: "v", Size: "50"},
		},
	},
	"frontend": {
		Name: "frontend",
		Slots: []PaneSlot{
			{DefaultRole: "editor", DefaultTitle: "editor", SplitFrom: -1},
			{DefaultRole: "shell", DefaultTitle: "shell", SplitFrom: 0, Direction: "h", Size: "42"},
			{DefaultRole: "server", DefaultTitle: "server", SplitFrom: 1, Direction: "v", Size: "45"},
			{DefaultRole: "tests", DefaultTitle: "tests", SplitFrom: 2, Direction: "v", Size: "50"},
			{DefaultRole: "logs", DefaultTitle: "logs", SplitFrom: 3, Direction: "v", Size: "50"},
		},
	},
	"ops": {
		Name: "ops",
		Slots: []PaneSlot{
			{DefaultRole: "shell", DefaultTitle: "shell", SplitFrom: -1},
			{DefaultRole: "server", DefaultTitle: "service", SplitFrom: 0, Direction: "h", Size: "50"},
			{DefaultRole: "logs", DefaultTitle: "logs", SplitFrom: 1, Direction: "v", Size: "50"},
			{DefaultRole: "docs", DefaultTitle: "docs", SplitFrom: 2, Direction: "v", Size: "50"},
		},
	},
	"agent-lab": {
		Name: "agent-lab",
		Slots: []PaneSlot{
			{DefaultRole: "editor", DefaultTitle: "editor", SplitFrom: -1},
			{DefaultRole: "shell", DefaultTitle: "shell", SplitFrom: 0, Direction: "h", Size: "40"},
			{DefaultRole: "tests", DefaultTitle: "tests", SplitFrom: 1, Direction: "v", Size: "34"},
			{DefaultRole: "logs", DefaultTitle: "logs", SplitFrom: 2, Direction: "v", Size: "50"},
			{DefaultRole: "agent", DefaultTitle: "agent", SplitFrom: 0, Direction: "v", Size: "28"},
			{DefaultRole: "docs", DefaultTitle: "docs", SplitFrom: 4, Direction: "h", Size: "45"},
		},
	},
}

func presetFor(name string) (LayoutPreset, error) {
	preset, ok := presets[name]
	if !ok {
		return LayoutPreset{}, fmt.Errorf("unknown layout %q", name)
	}
	return preset, nil
}

func expandWindow(window manifest.Window) ([]PanePlan, error) {
	preset, err := presetFor(window.Layout)
	if err != nil {
		return nil, err
	}
	if len(window.Panes) > len(preset.Slots) {
		return nil, fmt.Errorf("window %q layout %q supports %d panes, got %d", window.Name, window.Layout, len(preset.Slots), len(window.Panes))
	}

	plans := make([]PanePlan, 0, len(window.Panes))
	for i, pane := range window.Panes {
		slot := preset.Slots[i]
		if pane.Role == "" {
			pane.Role = slot.DefaultRole
		}
		if pane.Title == "" {
			pane.Title = slot.DefaultTitle
		}
		plans = append(plans, PanePlan{
			Pane:      pane,
			SplitFrom: slot.SplitFrom,
			Direction: slot.Direction,
			Size:      slot.Size,
		})
	}
	return plans, nil
}
