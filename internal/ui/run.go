package ui

import (
	"fmt"
	"os"
	"runtime"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nobarudo/gomad/internal/sideber"
	"github.com/nobarudo/gomad/internal/watcher"
	"github.com/nobarudo/gomad/themes"
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

// RunFile はローカルファイルまたはディレクトリを読み込んでビューアを起動します
func RunFile(filePath string, style string, watch bool) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to access path: %w", err)
	}

	var (
		targetFile     = filePath
		dirPath        string
		dirFiles       []string
		fileIndex      int
		showFilePicker bool
	)

	if info.IsDir() {
		files, readmeIdx, err := findMarkdownFiles(filePath)
		if err != nil {
			return err
		}
		dirPath = filePath
		dirFiles = files
		fileIndex = readmeIdx
		targetFile = files[readmeIdx]
		showFilePicker = true
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	styles := themes.AvailableStyles()

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
		w, watchErr = watcher.New(targetFile)
		if watchErr == nil {
			reloadStatus = "⚡ Auto-reload"
		}
	}

	m := model{
		filePath:        targetFile,
		content:         string(content),
		currentStyle:    style,
		availableStyles: styles,
		styleIndex:      initialIndex,
		searchInput:     ti,
		sidebar:         sideber.New(),
		watcher:         w,
		reloadStatus:    reloadStatus,
		isStdin:         false,
		dirPath:         dirPath,
		dirFiles:        dirFiles,
		fileIndex:       fileIndex,
		showFilePicker:  showFilePicker,
		watchEnabled:    watch,
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

	styles := themes.AvailableStyles()

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
	if finalM, ok := finalModel.(model); ok {
		if finalM.watcher != nil {
			_ = finalM.watcher.Close()
		}
		if err == nil && finalM.err != nil {
			return finalM.err
		}
	}
	if err != nil {
		return err
	}

	return nil
}
