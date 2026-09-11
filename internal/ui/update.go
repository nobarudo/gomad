package ui

import (
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
