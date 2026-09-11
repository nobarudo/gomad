package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"gomad/internal/sideber"
)

// Run はUIを初期化してプログラムを開始します
func Run(filePath string, style string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	styles := []string{"tokyo-night", "dracula", "dark", "light", "pink", "notty"}

	initialIndex := 0
	for i, s := range styles {
		if s == style {
			initialIndex = i
			break
		}
	}

	ti := textinput.New()
	ti.Placeholder = "Search... (Enter to search, Esc to cancel)"
	ti.Prompt = "/"
	ti.CharLimit = 100
	ti.Width = 40

	m := model{
		filePath:        filePath,
		content:         string(content),
		currentStyle:    style,
		availableStyles: styles,
		styleIndex:      initialIndex,
		searchInput:     ti,
		sidebar:         sideber.New(),
	}

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if finalM, ok := finalModel.(model); ok && finalM.err != nil {
		return finalM.err
	}

	return nil
}
