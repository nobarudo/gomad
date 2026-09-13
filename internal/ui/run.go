package ui

import (
	"fmt"
	"os"
	"runtime"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nobarudo/gomad/internal/sideber"
	"github.com/nobarudo/gomad/internal/watcher"
)

// openTTY はパイプ入力時にキーボード入力を受け付けるためのTTYデバイスを開きます
func openTTY() (*os.File, error) {
	if runtime.GOOS == "windows" {
		return os.Open("CONIN$")
	}
	return os.Open("/dev/tty")
}

// Run は後方互換性のための関数です
func Run(filePath string, style string, watch bool) error {
	return RunFile(filePath, style, watch)
}

// RunFile はローカルファイルを読み込んでビューアを起動します
func RunFile(filePath string, style string, watch bool) error {
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

	var (
		w            *watcher.Watcher
		reloadStatus string
	)
	if watch {
		var watchErr error
		w, watchErr = watcher.New(filePath)
		if watchErr == nil {
			defer w.Close()
			reloadStatus = "⚡ Auto-reload"
		}
	}

	m := model{
		filePath:        filePath,
		content:         string(content),
		currentStyle:    style,
		availableStyles: styles,
		styleIndex:      initialIndex,
		searchInput:     ti,
		sidebar:         sideber.New(),
		watcher:         w,
		reloadStatus:    reloadStatus,
		isStdin:         false,
	}

	return runProgram(m, nil)
}

// RunStdin は標準入力（パイプ）から渡されたテキストを表示します
func RunStdin(content string, style string) error {
	tty, err := openTTY()
	if err != nil {
		return fmt.Errorf("failed to open terminal for keyboard input: %w", err)
	}
	defer tty.Close()

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
		filePath:        "(stdin)",
		content:         content,
		currentStyle:    style,
		availableStyles: styles,
		styleIndex:      initialIndex,
		searchInput:     ti,
		sidebar:         sideber.New(),
		isStdin:         true,
	}

	return runProgram(m, tty)
}

func runProgram(m model, tty *os.File) error {
	opts := []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	}
	if tty != nil {
		opts = append(opts, tea.WithInput(tty))
	}

	p := tea.NewProgram(m, opts...)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if finalM, ok := finalModel.(model); ok && finalM.err != nil {
		return finalM.err
	}

	return nil
}
