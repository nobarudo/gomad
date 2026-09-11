package ui

import "fmt"

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	if !m.ready {
		return "\n  Initializing..."
	}

	// モーダルの描画優先順位
	if m.showHelpModal {
		return m.helpModalView()
	}
	if m.showStylePicker {
		return m.stylePickerView()
	}

	// 上部ガイド：シンプルに保ち「:」でキーバインド一覧を表示することを案内
	header := fmt.Sprintf("📖 %s  [Style: %s] ('s':スタイル変更 | ':':キーバインド | 'q':QUIT)", m.filePath, m.currentStyle)

	var footer string
	if m.showSearchInput {
		footer = " " + m.searchInput.View()
	} else {
		scrollPct := m.viewport.ScrollPercent() * 100
		searchStatus := ""
		if m.searchQuery != "" {
			if len(m.searchResults) > 0 {
				searchStatus = fmt.Sprintf(" | 🔍 %q (%d/%d) [n/N: Next/Prev, Esc: Clear]", m.searchQuery, m.currentMatchIndex+1, len(m.searchResults))
			} else {
				searchStatus = fmt.Sprintf(" | 🔍 %q (No matches) [Esc: Clear]", m.searchQuery)
			}
		}
		footer = fmt.Sprintf(" Scroll: %3.f%%%s", scrollPct, searchStatus)
	}

	return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer)
}
