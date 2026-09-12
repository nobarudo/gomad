package sideber

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Model は目次サイドバーのTUI状態を管理します
type Model struct {
	items   []Item
	cursor  int
	width   int
	height  int
	yOffset int
	focused bool
}

// New はModelの新しいインスタンスを生成します
func New() Model {
	return Model{
		items:   nil,
		cursor:  0,
		width:   30,
		height:  20,
		focused: true,
	}
}

// SetFocused はサイドバーのフォーカス状態を設定します
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// IsFocused はサイドバーにフォーカスがあるかを返します
func (m *Model) IsFocused() bool {
	return m.focused
}

// SetItems は目次項目をセットし、カーソル位置とオフセットを調整します
func (m *Model) SetItems(items []Item) {
	m.items = items
	if m.cursor >= len(m.items) {
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
		} else {
			m.cursor = 0
		}
	}
	m.adjustOffset()
}

// Items は現在の目次項目一覧を返します
func (m *Model) Items() []Item {
	return m.items
}

// Cursor は現在のカーソルインデックスを返します
func (m *Model) Cursor() int {
	return m.cursor
}

// SetCursor はカーソル位置を明示的に設定します
func (m *Model) SetCursor(cursor int) {
	if cursor < 0 {
		cursor = 0
	}
	if len(m.items) > 0 && cursor >= len(m.items) {
		cursor = len(m.items) - 1
	}
	m.cursor = cursor
	m.adjustOffset()
}

// SelectedItem は現在選択されている目次項目を返します
func (m *Model) SelectedItem() *Item {
	if len(m.items) == 0 || m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	return &m.items[m.cursor]
}

// SetSize はサイドバーの幅と高さを設定します
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.adjustOffset()
}

// MoveDown はカーソルを1つ下に移動します
func (m *Model) MoveDown() {
	if len(m.items) == 0 {
		return
	}
	if m.cursor < len(m.items)-1 {
		m.cursor++
		m.adjustOffset()
	}
}

// MoveUp はカーソルを1つ上に移動します
func (m *Model) MoveUp() {
	if len(m.items) == 0 {
		return
	}
	if m.cursor > 0 {
		m.cursor--
		m.adjustOffset()
	}
}

// GotoTop はカーソルを先頭へ移動します
func (m *Model) GotoTop() {
	m.cursor = 0
	m.adjustOffset()
}

// GotoBottom はカーソルを末尾へ移動します
func (m *Model) GotoBottom() {
	if len(m.items) > 0 {
		m.cursor = len(m.items) - 1
		m.adjustOffset()
	}
}

// HalfPageDown は半ページ分カーソルを下に移動します
func (m *Model) HalfPageDown() {
	visible := m.visibleLines()
	step := visible / 2
	if step < 1 {
		step = 1
	}
	m.cursor += step
	if m.cursor >= len(m.items) {
		m.cursor = max(0, len(m.items)-1)
	}
	m.adjustOffset()
}

// HalfPageUp は半ページ分カーソルを上に移動します
func (m *Model) HalfPageUp() {
	visible := m.visibleLines()
	step := visible / 2
	if step < 1 {
		step = 1
	}
	m.cursor -= step
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.adjustOffset()
}

// visibleLines は目次リストを描画可能な行数を返します
func (m *Model) visibleLines() int {
	// ヘッダー(2行) + フッター(2行) = 4行
	h := m.height - 4
	if h < 1 {
		return 1
	}
	return h
}

// adjustOffset はカーソルが画面外に出ないようにスクロール位置を調整します
func (m *Model) adjustOffset() {
	visible := m.visibleLines()
	if visible <= 0 {
		return
	}
	if m.cursor < m.yOffset {
		m.yOffset = m.cursor
	} else if m.cursor >= m.yOffset+visible {
		m.yOffset = m.cursor - visible + 1
	}
	if m.yOffset < 0 {
		m.yOffset = 0
	}
}

func truncateString(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// View はサイドバーの表示文字列をレンダリングします
func (m Model) View() string {
	contentWidth := m.width - 2 // 右枠線(1) + 余裕(1)
	if contentWidth < 5 {
		contentWidth = 5
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")) // シアン系

	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")) // 暗いグレー

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")) // マゼンタ / ハイライト

	if !m.focused {
		selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("246")) // 非フォーカス時はグレー
	}

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")) // 明るいグレー

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")) // ガイド文言

	var lines []string

	// 1. ヘッダー
	title := "📑 目次 (TOC)"
	lines = append(lines, headerStyle.Render(truncateString(title, contentWidth)))
	lines = append(lines, sepStyle.Render(strings.Repeat("─", contentWidth)))

	// 2. 目次アイテム一覧
	visible := m.visibleLines()
	if len(m.items) == 0 {
		emptyMsg := " (見出しなし)"
		lines = append(lines, sepStyle.Render(truncateString(emptyMsg, contentWidth)))
		for len(lines) < 2+visible {
			lines = append(lines, "")
		}
	} else {
		end := m.yOffset + visible
		if end > len(m.items) {
			end = len(m.items)
		}

		for i := m.yOffset; i < end; i++ {
			item := m.items[i]
			// # の数に応じたインデント (レベル1: 0スペース, レベル2: 2スペース, レベル3: 4スペース...)
			indent := strings.Repeat("  ", item.Level-1)
			hashes := strings.Repeat("#", item.Level)

			var lineContent string
			if i == m.cursor {
				prefix := "❯ "
				raw := fmt.Sprintf("%s%s%s %s", prefix, indent, hashes, item.Title)
				lineContent = selectedStyle.Render(truncateString(raw, contentWidth))
			} else {
				prefix := "  "
				raw := fmt.Sprintf("%s%s%s %s", prefix, indent, hashes, item.Title)
				lineContent = normalStyle.Render(truncateString(raw, contentWidth))
			}

			lines = append(lines, lineContent)
		}

		// 余白行の埋め合わせ
		for len(lines) < 2+visible {
			lines = append(lines, "")
		}
	}

	// 3. フッター（キーバインド操作ガイド）
	lines = append(lines, sepStyle.Render(strings.Repeat("─", contentWidth)))
	var guide string
	if m.focused {
		if contentWidth >= 28 {
			guide = "[j/k:移動 Enter:飛ぶ l:本文]"
		} else if contentWidth >= 20 {
			guide = "[j/k Enter l:本文]"
		} else {
			guide = "[Enter l:本文]"
		}
	} else {
		if contentWidth >= 20 {
			guide = "[h:目次操作 t:閉じる]"
		} else {
			guide = "[h:目次]"
		}
	}
	lines = append(lines, footerStyle.Render(truncateString(guide, contentWidth)))

	// サイドバー外枠（右側のみ縦線で仕切る）
	content := strings.Join(lines, "\n")

	borderWidth := m.width - 1
	if borderWidth < 1 {
		borderWidth = 1
	}

	borderColor := "63"
	if !m.focused {
		borderColor = "240"
	}

	sidebarStyle := lipgloss.NewStyle().
		Width(borderWidth).
		Height(m.height).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color(borderColor))

	return sidebarStyle.Render(content)
}
