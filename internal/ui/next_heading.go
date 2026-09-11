package ui

// 次の見出しへ移動
func (m *model) nextHeading() {
	current := m.viewport.YOffset
	for _, line := range m.headingLines {
		if line > current {
			m.viewport.SetYOffset(line)
			return
		}
	}
}
