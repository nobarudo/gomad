package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

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
	styleIndex      int
	availableStyles []string
}

// Run はUIを初期化してプログラムを開始します
func Run(filePath string, style string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	styles := []string{"tokyo-night", "dracula", "dark", "light", "pink", "notty"}

	// 指定されたスタイルのインデックスを探す
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

	_, err = p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

// Markdownコンテンツを指定スタイルでGlamourレンダリングする
func (m *model) renderContent() error {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(m.currentStyle),
		glamour.WithWordWrap(m.viewport.Width),
	)
	if err != nil {
		return err
	}

	rendered, err := renderer.Render(m.content)
	if err != nil {
		return err
	}

	m.viewport.SetContent(rendered)
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// --- テーマ選択モーダル表示中の操作 ---
		if m.showStylePicker {
			switch msg.String() {
			case "j", "down":
				m.styleIndex = (m.styleIndex + 1) % len(m.availableStyles)
			case "k", "up":
				m.styleIndex = (m.styleIndex - 1 + len(m.availableStyles)) % len(m.availableStyles)
			case "enter":
				// 新しいテーマを反映して再描画
				m.currentStyle = m.availableStyles[m.styleIndex]
				if err := m.renderContent(); err != nil {
					m.err = err
					return m, tea.Quit
				}
				m.showStylePicker = false
			case "esc", "s", "q":
				// キャンセルして閉じる
				m.showStylePicker = false
			}
			return m, nil
		}

		// --- 通常閲覧モード中の操作 ---
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "s":
			// テーマ選択モーダルを開く
			m.showStylePicker = true
			return m, nil

		// Vim操作
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

		// サイズに合わせて再レンダリング
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

	// テーマ選択モーダルの描画
	if m.showStylePicker {
		return m.stylePickerView()
	}

	header := fmt.Sprintf("📖 %s  [Style: %s] ('s':Theme | 'q':Quit)", m.filePath, m.currentStyle)
	footer := fmt.Sprintf(" Scroll: %3.f%% | Vim: j/k d/u f/b g/G", m.viewport.ScrollPercent()*100)

	return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer)
}

// テーマ選択用ポップアップ（モーダル）のUIを生成
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

	// モーダルを枠線で囲み、画面中央に配置
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
