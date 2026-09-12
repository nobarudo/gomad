package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// キーバインドヘルプモーダル
func (m model) helpModalView() string {
	var b strings.Builder
	b.WriteString("⌨️  Keybindings Help\n\n")

	b.WriteString("\x1b[1m[Scroll]\x1b[0m\n")
	b.WriteString("  j / k         : 1行移動 (Down/Up)\n")
	b.WriteString("  d / u         : 半ページ移動\n")
	b.WriteString("  f / b         : 1ページ移動\n")
	b.WriteString("  g / G         : 先頭 / 末尾へジャンプ\n\n")

	b.WriteString("\x1b[1m[Jump]\x1b[0m\n")
	b.WriteString("  t             : 目次（サイドバー）の開閉\n")
	b.WriteString("  h / l         : 目次 / 本文のフォーカス移動\n")
	b.WriteString("  Tab           : 目次と本文の操作フォーカス切り替え\n")
	b.WriteString("  { / }         : 前 / 次の段落（空行）へジャンプ\n")
	b.WriteString("  [ / ]         : 前 / 次の見出しへジャンプ\n\n")

	b.WriteString("\x1b[1m[Search]\x1b[0m\n")
	b.WriteString("  /             : 検索ワード入力\n")
	b.WriteString("  n / N         : 次 / 前の検索マッチへ移動\n")
	b.WriteString("  Esc           : 検索ハイライトをクリア\n\n")

	b.WriteString("\x1b[1m[System]\x1b[0m\n")
	b.WriteString("  s             : テーマ切り替えモーダル\n")
	b.WriteString("  r             : ファイルを手動再読み込み (Reload)\n")
	b.WriteString("  :             : このヘルプを表示\n")
	b.WriteString("  q / Ctrl+c    : 終了\n\n")

	b.WriteString("\x1b[90m[Esc / : / q / Enter で閉じる]\x1b[0m")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 3).
		Width(50)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		boxStyle.Render(b.String()),
	)
}
