package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/73ai/openbotkit/settings"
)

func Run(svc *settings.Service) error {
	for {
		m := newModel(svc)
		p := tea.NewProgram(m, tea.WithAltScreen())
		result, err := p.Run()
		if err != nil {
			return fmt.Errorf("settings TUI: %w", err)
		}

		rm, ok := result.(model)
		if !ok || rm.runWizard == nil {
			return nil
		}

		// Run the wizard outside bubbletea (huh forms need raw terminal).
		msg, err := rm.runWizard.EditFunc(svc)
		if err != nil {
			fmt.Printf("\n  Error: %v\n\n", err)
			fmt.Print("  Press Enter to continue...")
			fmt.Scanln()
		} else if msg != "" {
			fmt.Printf("\n  %s\n\n", msg)
			fmt.Print("  Press Enter to continue...")
			fmt.Scanln()
		}

		// Rebuild tree and restart TUI with updated config.
		svc.RebuildTree()
	}
}
