package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

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
	if m.showFilePicker {
		return m.filePickerView()
	}
	if m.showStylePicker {
		return m.stylePickerView()
	}

	// 上部ガイド：シンプルに保ち「:」でキーバインド一覧を表示することを案内
	reloadGuide := " | 'e':編集 | 'r':リロード"
	if m.isStdin {
		reloadGuide = ""
	}
	if len(m.dirFiles) > 0 {
		reloadGuide = " | 'p':一覧" + reloadGuide
	}
	header := fmt.Sprintf("📖 %s  [Style: %s] ('s':テーマ | 't':目次%s | ':':ヘルプ | 'q':QUIT)", m.filePath, m.currentStyle, reloadGuide)
	if m.showSidebar {
		if m.sidebarFocused {
			header = fmt.Sprintf("📖 %s  [目次操作中] (l/Tab:本文スクロール | t:目次を閉じる | ':':ヘルプ)", m.filePath)
		} else {
			header = fmt.Sprintf("📖 %s  [本文スクロール中] (h/Tab:目次操作 | t:目次を閉じる | ':':ヘルプ)", m.filePath)
		}
	}

	var body string
	if m.showSidebar {
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.sidebar.View(), m.viewport.View())
	} else {
		body = m.viewport.View()
	}

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
		linkStatus := ""
		if m.linkFocusActive && len(m.links) > 0 && m.currentLinkIndex >= 0 && m.currentLinkIndex < len(m.links) {
			activeLink := m.links[m.currentLinkIndex]
			backHint := ""
			if len(m.history) > 0 {
				backHint = ", BS: Back"
			}
			linkStatus = fmt.Sprintf(" | 🔗 (%d/%d) → %s [o/O: Next/Prev, Enter: Open%s, Esc: Clear]", m.currentLinkIndex+1, len(m.links), activeLink.url, backHint)
		} else if len(m.history) > 0 {
			linkStatus = " | [BS/Ctrl+o: 前のファイルへ戻る]"
		}
		reloadStatus := ""
		if m.reloadStatus != "" {
			reloadStatus = " | " + m.reloadStatus
		}
		footer = fmt.Sprintf(" Scroll: %3.f%%%s%s%s", scrollPct, reloadStatus, searchStatus, linkStatus)
	}

	return fmt.Sprintf("%s\n%s\n%s", header, body, footer)
}
