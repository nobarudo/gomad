package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nobarudo/gomad/internal/watcher"
)

// findMarkdownFiles は指定ディレクトリ直下のマークダウンファイル一覧と、
// READMEファイルが存在する場合のインデックス（なければ 0）を返します。
func findMarkdownFiles(dirPath string) ([]string, int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown") {
			files = append(files, filepath.Join(dirPath, name))
		}
	}

	if len(files) == 0 {
		return nil, 0, fmt.Errorf("no markdown files found in %s", dirPath)
	}

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(filepath.Base(files[i])) < strings.ToLower(filepath.Base(files[j]))
	})

	readmeIndex := 0
	for i, f := range files {
		base := strings.ToLower(filepath.Base(f))
		if base == "readme.md" || base == "readme.markdown" {
			readmeIndex = i
			break
		}
	}

	return files, readmeIndex, nil
}

// selectDirFile はディレクトリ内の指定インデックスのマークダウンファイルを開きます
func (m *model) selectDirFile(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.dirFiles) {
		return nil
	}

	targetPath := m.dirFiles[idx]
	data, err := os.ReadFile(targetPath)
	if err != nil {
		m.reloadStatus = fmt.Sprintf("⚠️ Failed to read %s", filepath.Base(targetPath))
		return nil
	}

	m.fileIndex = idx
	m.filePath = targetPath
	m.content = string(data)
	m.showFilePicker = false
	m.linkFocusActive = false
	m.currentLinkIndex = 0
	m.searchQuery = ""
	m.searchResults = nil

	if err := m.renderContent(); err != nil {
		m.err = err
		return tea.Quit
	}
	m.viewport.GotoTop()
	if m.showSidebar && !m.sidebarFocused {
		m.syncSidebarCursor()
	}

	if m.watcher != nil {
		_ = m.watcher.Close()
		m.watcher = nil
	}
	if m.watchEnabled {
		if newW, wErr := watcher.New(targetPath); wErr == nil {
			m.watcher = newW
			m.reloadStatus = "⚡ Auto-reload"
			return m.waitForFileChange()
		}
	}

	m.reloadStatus = fmt.Sprintf("📂 %s", filepath.Base(targetPath))
	return nil
}

// ファイル選択モーダル
func (m model) filePickerView() string {
	var b strings.Builder
	dirLabel := m.dirPath
	if dirLabel == "" {
		dirLabel = "."
	}
	b.WriteString(fmt.Sprintf("📂 Select Markdown File (%s)\n\n", dirLabel))

	// 画面高さに応じて表示件数を制限（スクロールウィンドウ）
	maxVisible := m.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}
	if maxVisible > 15 {
		maxVisible = 15
	}

	start := 0
	end := len(m.dirFiles)
	if len(m.dirFiles) > maxVisible {
		start = m.fileIndex - maxVisible/2
		if start < 0 {
			start = 0
		}
		end = start + maxVisible
		if end > len(m.dirFiles) {
			end = len(m.dirFiles)
			start = end - maxVisible
		}
	}

	for i := start; i < end; i++ {
		f := m.dirFiles[i]
		name := filepath.Base(f)

		cursor := "  "
		if i == m.fileIndex {
			cursor = "❯ "
		}

		activeMark := ""
		if f == m.filePath {
			activeMark = " (active)"
		}

		if i == m.fileIndex {
			b.WriteString(fmt.Sprintf("%s\x1b[1;35m%s%s\x1b[0m\n", cursor, name, activeMark))
		} else {
			b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, name, activeMark))
		}
	}

	b.WriteString("\n\x1b[90m[j/k: Move | Enter: Open | Esc/p: Close]\x1b[0m")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 3).
		Width(52)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		boxStyle.Render(b.String()),
	)
}
