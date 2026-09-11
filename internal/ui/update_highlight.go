package ui

import "strings"

// 検索結果の特定行をハイライトしてビューポートに表示する
func (m *model) updateHighlight() {
	if len(m.pristineLines) == 0 {
		return
	}

	m.renderedLines = make([]string, len(m.pristineLines))
	copy(m.renderedLines, m.pristineLines)

	if len(m.searchResults) == 0 {
		m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
		return
	}

	// 1. 全てのマッチ箇所に対して通常のハイライト（柔らかい黄色背景）を適用する
	for i, line := range m.renderedLines {
		m.renderedLines[i] = highlightQueryInLine(line, m.searchQuery, "\x1b[48;5;229m\x1b[38;5;0m", "\x1b[39;49m")
	}

	if m.currentMatchIndex < 0 {
		m.currentMatchIndex = 0
	}
	if m.currentMatchIndex >= len(m.searchResults) {
		m.currentMatchIndex = len(m.searchResults) - 1
	}

	targetLine := m.searchResults[m.currentMatchIndex]

	// 2. 現在のアクティブなマッチ箇所を目立つハイライト（オレンジ背景）にする
	m.renderedLines[targetLine] = highlightQueryInLine(m.pristineLines[targetLine], m.searchQuery, "\x1b[48;5;214m\x1b[38;5;0m", "\x1b[39;49m")

	m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))

	// 画面の中央に配置するように YOffset を設定する
	viewportHeight := m.viewport.Height
	yOffset := targetLine - (viewportHeight / 2)
	if yOffset < 0 {
		yOffset = 0
	}
	m.viewport.SetYOffset(yOffset)
}
