package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

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
	filePath        string
	content         string
	viewport        viewport.Model
	ready           bool
	err             error
	width           int
	height          int
	currentStyle    string
	showStylePicker bool
	showHelpModal   bool
	styleIndex      int
	availableStyles []string
	renderedLines   []string
	headingLines    []int
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

	m := model{
		filePath:        filePath,
		content:         string(content),
		currentStyle:    style,
		availableStyles: styles,
		styleIndex:      initialIndex,
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
	m.updateHeadingLines()

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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

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
	footer := fmt.Sprintf(" Scroll: %3.f%%", m.viewport.ScrollPercent()*100)

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
