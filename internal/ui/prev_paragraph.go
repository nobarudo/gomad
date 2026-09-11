package ui

import "strings"

// 前の空行（段落）へ移動
func (m *model) prevParagraph() {
	if len(m.renderedLines) == 0 {
		return
	}
	current := m.viewport.YOffset
	for i := current - 1; i >= 0; i-- {
		if strings.TrimSpace(stripANSI(m.renderedLines[i])) == "" {
			m.viewport.SetYOffset(i)
			return
		}
	}
	m.viewport.GotoTop()
}
