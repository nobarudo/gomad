package ui

import "strings"

// 検索結果の特定行をハイライトしてビューポートに表示する
func (m *model) updateHighlight() {
	if len(m.pristineLines) == 0 {
		return
	}

	if len(m.renderedLines) != len(m.pristineLines) {
		m.renderedLines = make([]string, len(m.pristineLines))
	}
	copy(m.renderedLines, m.pristineLines)

	if len(m.searchResults) == 0 {
		m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
		return
	}

	if m.currentMatchIndex < 0 {
		m.currentMatchIndex = 0
	}
	if m.currentMatchIndex >= len(m.searchResults) {
		m.currentMatchIndex = len(m.searchResults) - 1
	}

	targetLine := m.searchResults[m.currentMatchIndex]

	// マッチした行だけを対象にハイライトを適用する（全行スキャンを回避）
	for _, lineIdx := range m.searchResults {
		if lineIdx < 0 || lineIdx >= len(m.pristineLines) {
			continue
		}
		if lineIdx == targetLine {
			// アクティブなマッチ箇所: 目立つオレンジ背景
			m.renderedLines[lineIdx] = highlightQueryInLine(m.pristineLines[lineIdx], m.searchQuery, "\x1b[48;5;214m\x1b[38;5;0m", "\x1b[39;49m")
		} else {
			// その他のマッチ箇所: 柔らかい黄色背景
			m.renderedLines[lineIdx] = highlightQueryInLine(m.pristineLines[lineIdx], m.searchQuery, "\x1b[48;5;229m\x1b[38;5;0m", "\x1b[39;49m")
		}
	}

	m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))

	// 画面の中央に配置するように YOffset を設定する
	viewportHeight := m.viewport.Height
	yOffset := targetLine - (viewportHeight / 2)
	if yOffset < 0 {
		yOffset = 0
	}
	m.viewport.SetYOffset(yOffset)
}
