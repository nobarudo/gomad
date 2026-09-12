package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"gomad/internal/watcher"
)

type fileReloadMsg struct {
	event watcher.Event
}

func (m model) Init() tea.Cmd {
	return m.waitForFileChange()
}

func (m model) waitForFileChange() tea.Cmd {
	if m.watcher == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-m.watcher.Events()
		if !ok {
			return nil
		}
		return fileReloadMsg{event: event}
	}
}
