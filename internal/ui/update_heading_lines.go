package ui

import (
	"gomad/internal/sideber"
)

// Markdownの見出し位置（行番号）を解析・保持する
func (m *model) updateHeadingLines() {
	items := sideber.ParseItems(m.content, m.renderedLines)
	m.sidebar.SetItems(items)

	hLines := make([]int, len(items))
	for i, item := range items {
		hLines[i] = item.Line
	}
	m.headingLines = hLines
}
