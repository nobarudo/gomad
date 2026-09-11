package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// テーマ選択モーダル
func (m model) stylePickerView() string {
	var b strings.Builder
	b.WriteString("🎨 Select Color Theme\n\n")

	for i, st := range m.availableStyles {
		cursor := "  "
		if i == m.styleIndex {
			cursor = "❯ "
		}

		activeMark := ""
		if st == m.currentStyle {
			activeMark = " (active)"
		}

		if i == m.styleIndex {
			b.WriteString(fmt.Sprintf("%s\x1b[1;35m%s%s\x1b[0m\n", cursor, st, activeMark))
		} else {
			b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, st, activeMark))
		}
	}

	b.WriteString("\n\x1b[90m[j/k: Move | Enter: Select | Esc/s: Cancel]\x1b[0m")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 3).
		Width(45)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		boxStyle.Render(b.String()),
	)
}
