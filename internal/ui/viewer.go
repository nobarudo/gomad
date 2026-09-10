package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

type model struct {
	filePath string
	content  string
	viewport viewport.Model
	ready    bool
	err      error
}

// Run はUIを初期化してプログラムを開始します
func Run(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	m := model{
		filePath: filePath,
		content:  string(content),
	}

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // フルスクリーンモード
		tea.WithMouseCellMotion(), // マウススクロール有効化
	)

	_, err = p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := 1
		footerHeight := 1
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			renderer, err := glamour.NewTermRenderer(
				glamour.WithStandardStyle("dark"),
				glamour.WithWordWrap(msg.Width),
			)
			if err != nil {
				m.err = err
				return m, tea.Quit
			}

			rendered, err := renderer.Render(m.content)
			if err != nil {
				m.err = err
				return m, tea.Quit
			}

			// Viewport（表示領域）の初期化
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(rendered)
			m.ready = true
		} else {
			// ターミナルサイズが変更された場合
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	// キー入力（スクロールなど）をViewportに転送
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

	header := fmt.Sprintf("📖 %s (Press 'q' to quit)", m.filePath)
	footer := fmt.Sprintf(" Scroll: %3.f%%", m.viewport.ScrollPercent()*100)

	return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer)
}
