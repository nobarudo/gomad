package ui

import "sort"

// 次の見出しへ移動
func (m *model) nextHeading() {
	current := m.viewport.YOffset
	idx := sort.Search(len(m.headingLines), func(i int) bool {
		return m.headingLines[i] > current
	})
	if idx < len(m.headingLines) {
		m.viewport.SetYOffset(m.headingLines[idx])
	}
}
