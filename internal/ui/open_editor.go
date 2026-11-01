package ui

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type editorFinishedMsg struct {
	err error
}

// resolveEditorCmd は環境変数 ($VISUAL -> $EDITOR) またはOSデフォルトから
// 起動するエディタの実行ファイル名と引数リストを返します。
func resolveEditorCmd(filePath string) (string, []string) {
	editor := strings.TrimSpace(os.Getenv("VISUAL"))
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}

	if editor != "" {
		fields := strings.Fields(editor)
		if len(fields) > 0 {
			args := append(fields[1:], filePath)
			return fields[0], args
		}
	}

	if runtime.GOOS == "windows" {
		return "notepad", []string{filePath}
	}
	return "vi", []string{filePath}
}

// openEditor は外部エディタを起動する tea.Cmd を返します。
func (m model) openEditor() tea.Cmd {
	bin, args := resolveEditorCmd(m.filePath)
	c := exec.Command(bin, args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}
