package ui

import "sort"

// 前の見出しへ移動
func (m *model) prevHeading() {
	current := m.viewport.YOffset
	idx := sort.Search(len(m.headingLines), func(i int) bool {
		return m.headingLines[i] >= current
	})
	if idx > 0 {
		m.viewport.SetYOffset(m.headingLines[idx-1])
	}
}
