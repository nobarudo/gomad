package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
)

// Markdownコンテンツを指定スタイルでGlamourレンダリングする
func (m *model) renderContent() error {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(m.currentStyle),
		glamour.WithWordWrap(m.viewport.Width),
	)
	if err != nil {
		renderer, err = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(m.viewport.Width),
		)
		if err != nil {
			return fmt.Errorf("failed to create glamour renderer: %w", err)
		}
	}

	rendered, err := renderer.Render(m.content)
	if err != nil {
		return fmt.Errorf("failed to render markdown: %w", err)
	}

	m.viewport.SetContent(rendered)
	m.renderedLines = strings.Split(rendered, "\n")
	m.pristineLines = make([]string, len(m.renderedLines))
	copy(m.pristineLines, m.renderedLines)
	m.updateHeadingLines()

	if m.searchQuery != "" {
		m.executeSearch(m.searchQuery)
	}

	return nil
}
