package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/73ai/openbotkit/settings"
)

func Run(svc *settings.Service) error {
	m := newModel(svc)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("settings TUI: %w", err)
	}
	return nil
}
