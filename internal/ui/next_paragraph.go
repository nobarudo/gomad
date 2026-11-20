package ui

import "strings"

// 次の空行（段落）へ移動
func (m *model) nextParagraph() {
	if len(m.renderedLines) == 0 {
		return
	}
	current := m.viewport.YOffset
	usePlain := len(m.plainLines) == len(m.renderedLines)
	for i := current + 1; i < len(m.renderedLines); i++ {
		var plain string
		if usePlain {
			plain = m.plainLines[i]
		} else {
			plain = stripANSI(m.renderedLines[i])
		}
		if strings.TrimSpace(plain) == "" {
			m.viewport.SetYOffset(i)
			return
		}
	}
	m.viewport.GotoBottom()
}
