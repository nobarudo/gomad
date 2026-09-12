package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

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
	case fileReloadMsg:
		if msg.event.Err != nil {
			m.reloadStatus = "⚠️ Reload failed"
			return m, m.waitForFileChange()
		}

		if err := m.applyNewContent(msg.event.Content); err != nil {
			m.err = err
			return m, tea.Quit
		}

		if !msg.event.ModTime.IsZero() {
			m.lastReloadTime = msg.event.ModTime
		} else {
			m.lastReloadTime = time.Now()
		}
		m.reloadStatus = fmt.Sprintf("⚡ %s", m.lastReloadTime.Format("15:04:05"))
		return m, m.waitForFileChange()

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

		// --- 目次サイドバー表示中の操作 ---
		if m.showSidebar {
			// Tab キーで目次サイドバーと本文の操作フォーカスを切り替え
			if msg.String() == "tab" {
				m.sidebarFocused = !m.sidebarFocused
				m.sidebar.SetFocused(m.sidebarFocused)
				return m, nil
			}

			// サイドバーにフォーカスがある時の操作
			if m.sidebarFocused {
				switch msg.String() {
				case "esc", "t", "q":
					m.showSidebar = false
					m.sidebarFocused = false
					m.sidebar.SetFocused(false)
					m.calculateLayout()
					if err := m.renderContent(); err != nil {
						m.err = err
						return m, tea.Quit
					}
					return m, nil

				case "l", "right":
					m.sidebarFocused = false
					m.sidebar.SetFocused(false)
					return m, nil

				case "j", "down":
					m.sidebar.MoveDown()
					return m, nil

				case "k", "up":
					m.sidebar.MoveUp()
					return m, nil

				case "g":
					m.sidebar.GotoTop()
					return m, nil

				case "G":
					m.sidebar.GotoBottom()
					return m, nil

				case "d", "ctrl+d":
					m.sidebar.HalfPageDown()
					return m, nil

				case "u", "ctrl+u":
					m.sidebar.HalfPageUp()
					return m, nil

				case "enter":
					// サイドバーを開いたまま、選択した見出しへ本文をジャンプスクロール
					cursor := m.sidebar.Cursor()
					if cursor >= 0 && cursor < len(m.headingLines) {
						m.viewport.SetYOffset(m.headingLines[cursor])
					}
					return m, nil
				}
				return m, nil
			}

			// 本文にフォーカスがある時、'h' または 'left' で目次サイドバーにフォーカスを移動
			if msg.String() == "h" || msg.String() == "left" {
				m.sidebarFocused = true
				m.sidebar.SetFocused(true)
				return m, nil
			}

			// 本文にフォーカスがある時、't' で目次を閉じる
			if msg.String() == "t" {
				m.showSidebar = false
				m.sidebarFocused = false
				m.sidebar.SetFocused(false)
				m.calculateLayout()
				if err := m.renderContent(); err != nil {
					m.err = err
					return m, tea.Quit
				}
				return m, nil
			}
			// それ以外のキー (j/k, d/u, /, n/N 等) は後続の通常閲覧モードで処理
		}

		// --- 通常閲覧モード中の操作 ---
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "s":
			m.showStylePicker = true
			return m, nil

		case "r":
			data, err := os.ReadFile(m.filePath)
			if err != nil {
				m.reloadStatus = "⚠️ Read failed"
				return m, nil
			}
			if err := m.applyNewContent(string(data)); err != nil {
				m.err = err
				return m, tea.Quit
			}
			m.lastReloadTime = time.Now()
			m.reloadStatus = fmt.Sprintf("⚡ %s", m.lastReloadTime.Format("15:04:05"))
			return m, nil

		case "t":
			m.showSidebar = true
			m.sidebarFocused = true
			m.sidebar.SetFocused(true)
			m.calculateLayout()
			if err := m.renderContent(); err != nil {
				m.err = err
				return m, tea.Quit
			}
			// 現在の閲覧位置に近い見出しにカーソルを合わせる
			currentY := m.viewport.YOffset
			closestIdx := 0
			for i, line := range m.headingLines {
				if line <= currentY {
					closestIdx = i
				} else {
					break
				}
			}
			m.sidebar.SetCursor(closestIdx)
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

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-2)
			m.viewport.YPosition = headerHeight
			m.ready = true
		}
		m.calculateLayout()

		if err := m.renderContent(); err != nil {
			m.err = err
			return m, tea.Quit
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// calculateLayout はサイドバーの開閉状態と画面サイズに応じてレイアウトを計算します
func (m *model) calculateLayout() {
	verticalMarginHeight := 2 // header(1) + footer(1)
	contentHeight := m.height - verticalMarginHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	if m.showSidebar {
		// 目次サイドバーの幅：画面幅の約30%、最小24、最大40
		sbWidth := 30
		if m.width < 80 {
			sbWidth = m.width * 35 / 100
			if sbWidth < 20 {
				sbWidth = 20
			}
		} else if m.width > 120 {
			sbWidth = 36
		}

		if m.width-sbWidth < 20 {
			sbWidth = m.width / 2
		}
		vpWidth := m.width - sbWidth
		if vpWidth < 10 {
			vpWidth = 10
		}

		m.sidebar.SetSize(sbWidth, contentHeight)
		m.viewport.Width = vpWidth
		m.viewport.Height = contentHeight
	} else {
		m.viewport.Width = m.width
		m.viewport.Height = contentHeight
	}
}

// applyNewContent はコンテンツを更新し、スクロール位置を適切に維持して再レンダリングします
func (m *model) applyNewContent(newContent string) error {
	m.content = newContent
	oldOffset := m.viewport.YOffset

	if err := m.renderContent(); err != nil {
		return err
	}

	maxOffset := len(m.renderedLines) - m.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if oldOffset > maxOffset {
		m.viewport.SetYOffset(maxOffset)
	} else {
		m.viewport.SetYOffset(oldOffset)
	}
	return nil
}
