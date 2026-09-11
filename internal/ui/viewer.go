package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// ANSIエスケープコードを除去するヘルパー
func stripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

type model struct {
	filePath          string
	content           string
	viewport          viewport.Model
	ready             bool
	err               error
	width             int
	height            int
	currentStyle      string
	showStylePicker   bool
	showHelpModal     bool
	styleIndex        int
	availableStyles   []string
	renderedLines     []string
	headingLines      []int
	searchInput       textinput.Model
	showSearchInput   bool
	searchQuery       string
	searchResults     []int
	currentMatchIndex int
	pristineLines     []string
}

// Run はUIを初期化してプログラムを開始します
func Run(filePath string, style string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	styles := []string{"tokyo-night", "dracula", "dark", "light", "pink", "notty"}

	initialIndex := 0
	for i, s := range styles {
		if s == style {
			initialIndex = i
			break
		}
	}

	ti := textinput.New()
	ti.Placeholder = "Search... (Enter to search, Esc to cancel)"
	ti.Prompt = "/"
	ti.CharLimit = 100
	ti.Width = 40

	m := model{
		filePath:        filePath,
		content:         string(content),
		currentStyle:    style,
		availableStyles: styles,
		styleIndex:      initialIndex,
		searchInput:     ti,
	}

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if finalM, ok := finalModel.(model); ok && finalM.err != nil {
		return finalM.err
	}

	return nil
}

func (m model) Init() tea.Cmd {
	return nil
}

// Markdownの見出し位置（行番号）を解析・保持する
func (m *model) updateHeadingLines() {
	re := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	matches := re.FindAllStringSubmatch(m.content, -1)

	var hLines []int
	searchStart := 0

	for _, match := range matches {
		title := strings.TrimSpace(match[2])
		for i := searchStart; i < len(m.renderedLines); i++ {
			if strings.Contains(m.renderedLines[i], title) {
				hLines = append(hLines, i)
				searchStart = i + 1
				break
			}
		}
	}
	m.headingLines = hLines
}

// Markdownコンテンツを指定スタイルでGlamourレンダリングする
func (m *model) renderContent() error {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(m.currentStyle),
		glamour.WithWordWrap(m.viewport.Width),
	)
	if err != nil {
		renderer, err = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(m.viewport.Width),
		)
		if err != nil {
			return fmt.Errorf("failed to create glamour renderer: %w", err)
		}
	}

	rendered, err := renderer.Render(m.content)
	if err != nil {
		return fmt.Errorf("failed to render markdown: %w", err)
	}

	m.viewport.SetContent(rendered)
	m.renderedLines = strings.Split(rendered, "\n")
	m.pristineLines = make([]string, len(m.renderedLines))
	copy(m.pristineLines, m.renderedLines)
	m.updateHeadingLines()

	if m.searchQuery != "" {
		m.executeSearch(m.searchQuery)
	}

	return nil
}

// 次の空行（段落）へ移動
func (m *model) nextParagraph() {
	if len(m.renderedLines) == 0 {
		return
	}
	current := m.viewport.YOffset
	for i := current + 1; i < len(m.renderedLines); i++ {
		if strings.TrimSpace(stripANSI(m.renderedLines[i])) == "" {
			m.viewport.SetYOffset(i)
			return
		}
	}
	m.viewport.GotoBottom()
}

// 前の空行（段落）へ移動
func (m *model) prevParagraph() {
	if len(m.renderedLines) == 0 {
		return
	}
	current := m.viewport.YOffset
	for i := current - 1; i >= 0; i-- {
		if strings.TrimSpace(stripANSI(m.renderedLines[i])) == "" {
			m.viewport.SetYOffset(i)
			return
		}
	}
	m.viewport.GotoTop()
}

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

type segment struct {
	text   string
	isANSI bool
}

// ANSIエスケープシーケンスと通常の文字に分解する（ルーン対応）
func parseSegments(ansiStr string) []segment {
	var segments []segment
	runes := []rune(ansiStr)
	n := len(runes)
	i := 0
	for i < n {
		if runes[i] == '\x1b' {
			start := i
			i++ // skip '\x1b'
			for i < n {
				r := runes[i]
				i++
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
					break
				}
			}
			segments = append(segments, segment{text: string(runes[start:i]), isANSI: true})
		} else {
			segments = append(segments, segment{text: string(runes[i]), isANSI: false})
			i++
		}
	}
	return segments
}

// 行内の検索ワードにのみハイライトを適用する（ルーンベースで完全にズレを防ぐ）
func highlightQueryInLine(ansiStr string, query string, highlightStart, highlightEnd string) string {
	if query == "" {
		return ansiStr
	}

	segments := parseSegments(ansiStr)

	var plainBuilder strings.Builder
	var plainToSeg []int

	for segIdx, seg := range segments {
		if !seg.isANSI {
			plainBuilder.WriteString(seg.text)
			plainToSeg = append(plainToSeg, segIdx)
		}
	}

	plainText := plainBuilder.String()
	plainRunes := []rune(plainText)
	lowerPlainRunes := []rune(strings.ToLower(plainText))
	lowerQueryRunes := []rune(strings.ToLower(query))

	queryLen := len(lowerQueryRunes)
	if queryLen == 0 || len(plainRunes) < queryLen {
		return ansiStr
	}

	// ルーンスライスでの検索
	var matchStarts []int
	n := len(lowerPlainRunes)
	m := len(lowerQueryRunes)
	for i := 0; i <= n-m; i++ {
		match := true
		for j := 0; j < m; j++ {
			if lowerPlainRunes[i+j] != lowerQueryRunes[j] {
				match = false
				break
			}
		}
		if match {
			matchStarts = append(matchStarts, i)
			i += m - 1 // 重複防止
		}
	}

	if len(matchStarts) == 0 {
		return ansiStr
	}

	for _, start := range matchStarts {
		segStart := plainToSeg[start]
		segEnd := plainToSeg[start+queryLen-1]

		segments[segStart].text = highlightStart + segments[segStart].text
		segments[segEnd].text = segments[segEnd].text + highlightEnd
	}

	var result strings.Builder
	for _, seg := range segments {
		result.WriteString(seg.text)
	}
	return result.String()
}

// 検索ワードで各行を検索する
func (m *model) executeSearch(query string) {
	m.searchQuery = query
	m.searchResults = nil
	m.currentMatchIndex = 0

	if query == "" {
		if len(m.pristineLines) > 0 {
			m.renderedLines = make([]string, len(m.pristineLines))
			copy(m.renderedLines, m.pristineLines)
			m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
		}
		return
	}

	lowerQuery := strings.ToLower(query)
	for i, line := range m.pristineLines {
		plain := strings.ToLower(stripANSI(line))
		if strings.Contains(plain, lowerQuery) {
			m.searchResults = append(m.searchResults, i)
		}
	}

	m.updateHighlight()
}

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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// --- 検索入力モード中の操作 ---
	if m.showSearchInput {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.showSearchInput = false
				m.searchInput.Blur()
				return m, nil
			case "enter":
				m.showSearchInput = false
				m.searchInput.Blur()
				query := m.searchInput.Value()
				m.executeSearch(query)
				return m, nil
			}
		}
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// --- キーバインドヘルプモーダル表示中の操作 ---
		if m.showHelpModal {
			switch msg.String() {
			case "esc", "q", ":", "enter":
				m.showHelpModal = false
			}
			return m, nil
		}

		// --- テーマ選択モーダル表示中の操作 ---
		if m.showStylePicker {
			switch msg.String() {
			case "j", "down":
				m.styleIndex = (m.styleIndex + 1) % len(m.availableStyles)
			case "k", "up":
				m.styleIndex = (m.styleIndex - 1 + len(m.availableStyles)) % len(m.availableStyles)
			case "enter":
				m.currentStyle = m.availableStyles[m.styleIndex]
				if err := m.renderContent(); err != nil {
					m.err = err
					return m, tea.Quit
				}
				m.showStylePicker = false
			case "esc", "s", "q":
				m.showStylePicker = false
			}
			return m, nil
		}

		// --- 通常閲覧モード中の操作 ---
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "s":
			m.showStylePicker = true
			return m, nil

		case ":":
			m.showHelpModal = true
			return m, nil

		case "/":
			m.showSearchInput = true
			m.searchInput.Focus()
			m.searchInput.SetValue("")
			return m, textinput.Blink

		case "n":
			if len(m.searchResults) > 0 {
				m.currentMatchIndex = (m.currentMatchIndex + 1) % len(m.searchResults)
				m.updateHighlight()
			}

		case "N":
			if len(m.searchResults) > 0 {
				m.currentMatchIndex = (m.currentMatchIndex - 1 + len(m.searchResults)) % len(m.searchResults)
				m.updateHighlight()
			}

		case "esc":
			m.searchQuery = ""
			m.searchResults = nil
			m.currentMatchIndex = 0
			m.executeSearch("")

		// Vimジャンプ操作
		case "}":
			m.nextParagraph()
		case "{":
			m.prevParagraph()
		case "]":
			m.nextHeading()
		case "[":
			m.prevHeading()

		// Vim基本スクロール操作
		case "j", "down":
			m.viewport.LineDown(1)
		case "k", "up":
			m.viewport.LineUp(1)
		case "d", "ctrl+d":
			m.viewport.HalfViewDown()
		case "u", "ctrl+u":
			m.viewport.HalfViewUp()
		case "f", "ctrl+f":
			m.viewport.ViewDown()
		case "b", "ctrl+b":
			m.viewport.ViewUp()
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 1
		footerHeight := 1
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		if err := m.renderContent(); err != nil {
			m.err = err
			return m, tea.Quit
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

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

// テーマ選択モーダル
func (m model) stylePickerView() string {
	var b strings.Builder
	b.WriteString("🎨 Select Color Theme\n\n")

	for i, st := range m.availableStyles {
		cursor := "  "
		if i == m.styleIndex {
			cursor = "❯ "
		}

		activeMark := ""
		if st == m.currentStyle {
			activeMark = " (active)"
		}

		if i == m.styleIndex {
			b.WriteString(fmt.Sprintf("%s\x1b[1;35m%s%s\x1b[0m\n", cursor, st, activeMark))
		} else {
			b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, st, activeMark))
		}
	}

	b.WriteString("\n\x1b[90m[j/k: Move | Enter: Select | Esc/s: Cancel]\x1b[0m")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 3).
		Width(45)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		boxStyle.Render(b.String()),
	)
}

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
	b.WriteString("  { / }         : 前 / 次の段落（空行）へジャンプ\n")
	b.WriteString("  [ / ]         : 前 / 次の見出しへジャンプ\n\n")

	b.WriteString("\x1b[1m[Search]\x1b[0m\n")
	b.WriteString("  /             : 検索ワード入力\n")
	b.WriteString("  n / N         : 次 / 前の検索マッチへ移動\n")
	b.WriteString("  Esc           : 検索ハイライトをクリア\n\n")

	b.WriteString("\x1b[1m[System]\x1b[0m\n")
	b.WriteString("  s             : テーマ切り替えモーダル\n")
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
