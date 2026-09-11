package ui

// 前の見出しへ移動
func (m *model) prevHeading() {
	current := m.viewport.YOffset
	for i := len(m.headingLines) - 1; i >= 0; i-- {
		line := m.headingLines[i]
		if line < current {
			m.viewport.SetYOffset(line)
			return
		}
	}
}
