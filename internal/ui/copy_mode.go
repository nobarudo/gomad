package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
)

// codeBlock はMarkdown内のコードブロックとそのレンダリング後行範囲を表します
type codeBlock struct {
	language  string
	content   string
	lineCount int
	startLine int
	endLine   int
}

// copyToClipboardFunc はクリップボード書き込み関数（テストで差し替え可能）
var copyToClipboardFunc = clipboard.WriteAll

type rawCodeBlock struct {
	language  string
	content   string
	lines     []string
	lineCount int
}

// extractRawCodeBlocks はMarkdownテキストからフェンス付きコードブロックを抽出します
func extractRawCodeBlocks(content string) []rawCodeBlock {
	lines := strings.Split(content, "\n")
	var blocks []rawCodeBlock

	inFence := false
	var fenceChar byte
	fenceLen := 0
	fenceIndent := 0
	lang := ""
	var buf []string

	for _, line := range lines {
		// 先頭の空白文字数を数える（リスト内コードブロック等のインデント対応）
		indent := 0
		for indent < len(line) && (line[indent] == ' ' || line[indent] == '\t') {
			indent++
		}
		trimmed := line[indent:]

		if !inFence {
			if len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
				ch := trimmed[0]
				n := 0
				for n < len(trimmed) && trimmed[n] == ch {
					n++
				}
				if n >= 3 {
					rest := strings.TrimSpace(trimmed[n:])
					// バッククォートのインフォ文字列にバッククォートが含まれる場合はフェンスではない
					if ch == '`' && strings.ContainsRune(rest, '`') {
						continue
					}
					inFence = true
					fenceChar = ch
					fenceLen = n
					fenceIndent = indent
					fields := strings.Fields(rest)
					if len(fields) > 0 {
						lang = fields[0]
					} else {
						lang = ""
					}
					buf = nil
					continue
				}
			}
			continue
		}

		// 閉じフェンス判定
		if len(trimmed) >= fenceLen && trimmed[0] == fenceChar {
			n := 0
			for n < len(trimmed) && trimmed[n] == fenceChar {
				n++
			}
			if n >= fenceLen && strings.TrimSpace(trimmed[n:]) == "" {
				blocks = append(blocks, rawCodeBlock{
					language:  lang,
					content:   strings.Join(buf, "\n"),
					lines:     buf,
					lineCount: len(buf),
				})
				inFence = false
				continue
			}
		}

		// リスト内などでフェンス自体がインデントされていた場合、その分だけ取り除く
		codeLine := line
		if fenceIndent > 0 && len(codeLine) >= fenceIndent {
			allSpace := true
			for i := 0; i < fenceIndent; i++ {
				if codeLine[i] != ' ' && codeLine[i] != '\t' {
					allSpace = false
					break
				}
			}
			if allSpace {
				codeLine = codeLine[fenceIndent:]
			}
		}
		buf = append(buf, codeLine)
	}

	return blocks
}

func matchCodeLineInPlain(plainLine string, trimmedCode string) bool {
	if trimmedCode == "" {
		return false
	}
	if strings.Contains(plainLine, trimmedCode) {
		return true
	}
	// Glamour の折り返しで1行が分割された場合、先頭部分での一致を試みる
	runes := []rune(trimmedCode)
	if len(runes) > 16 {
		prefix := string(runes[:16])
		if strings.Contains(plainLine, prefix) {
			return true
		}
	}
	return false
}

// parseCodeBlocksWithPlainLines はコードブロックを抽出し、レンダリング後の行範囲にマッピングします
func parseCodeBlocksWithPlainLines(content string, plainLines []string) []codeBlock {
	rawBlocks := extractRawCodeBlocks(content)
	if len(rawBlocks) == 0 || len(plainLines) == 0 {
		return nil
	}

	blocks := make([]codeBlock, 0, len(rawBlocks))
	searchLine := 0

	for _, rb := range rawBlocks {
		startLine := -1
		endLine := -1

		// コードブロック内の非空行を順番に照合して開始行・終了行を特定する
		curSearch := searchLine
		for _, cl := range rb.lines {
			trimmed := strings.TrimSpace(cl)
			if trimmed == "" {
				continue
			}
			for i := curSearch; i < len(plainLines); i++ {
				if matchCodeLineInPlain(plainLines[i], trimmed) {
					if startLine == -1 {
						startLine = i
					}
					endLine = i
					curSearch = i + 1
					break
				}
			}
		}

		if startLine == -1 {
			// 空のコードブロックや見つからなかった場合のフォールバック
			startLine = searchLine
			if startLine >= len(plainLines) {
				startLine = len(plainLines) - 1
			}
			endLine = startLine
		}

		blocks = append(blocks, codeBlock{
			language:  rb.language,
			content:   rb.content,
			lineCount: rb.lineCount,
			startLine: startLine,
			endLine:   endLine,
		})
		searchLine = endLine + 1
	}

	return blocks
}

// enterCopyMode は 'y' キーでコピーモードを開始し、現在画面内（または次）のコードブロックを選択します
func (m *model) enterCopyMode() {
	if len(m.codeBlocks) == 0 {
		m.copyMode = false
		m.reloadStatus = "📋 No code blocks"
		return
	}

	// 検索ハイライト・リンクフォーカスと競合しないようクリア
	if m.searchQuery != "" {
		m.searchQuery = ""
		m.searchResults = nil
		m.currentMatchIndex = 0
	}
	m.linkFocusActive = false

	m.copyMode = true

	// 現在の表示範囲内（endLine >= YOffset）にある最初のコードブロックを探す
	idx := 0
	for i, cb := range m.codeBlocks {
		if cb.endLine >= m.viewport.YOffset {
			idx = i
			break
		}
	}
	m.currentCodeIndex = idx
	m.updateCopyHighlight(true)
}

// nextCodeBlock はコピーモード中に次のコードブロックへ移動します
func (m *model) nextCodeBlock() {
	if len(m.codeBlocks) == 0 {
		return
	}
	m.currentCodeIndex = (m.currentCodeIndex + 1) % len(m.codeBlocks)
	m.updateCopyHighlight(true)
}

// prevCodeBlock はコピーモード中に前のコードブロックへ移動します
func (m *model) prevCodeBlock() {
	if len(m.codeBlocks) == 0 {
		return
	}
	m.currentCodeIndex = (m.currentCodeIndex - 1 + len(m.codeBlocks)) % len(m.codeBlocks)
	m.updateCopyHighlight(true)
}

// exitCopyMode はコピーモードを終了し、通常表示に戻します
func (m *model) exitCopyMode() {
	if !m.copyMode {
		return
	}
	m.copyMode = false
	if len(m.pristineLines) > 0 {
		if len(m.renderedLines) != len(m.pristineLines) {
			m.renderedLines = make([]string, len(m.pristineLines))
		}
		copy(m.renderedLines, m.pristineLines)
		m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
	}
}

// yankSelectedCodeBlock は選択中のコードブロックをクリップボードにコピーし、通常モードへ戻ります
func (m *model) yankSelectedCodeBlock() {
	if !m.copyMode || len(m.codeBlocks) == 0 || m.currentCodeIndex < 0 || m.currentCodeIndex >= len(m.codeBlocks) {
		m.exitCopyMode()
		return
	}

	cb := m.codeBlocks[m.currentCodeIndex]
	err := copyToClipboardFunc(cb.content)
	m.exitCopyMode()

	if err != nil {
		m.reloadStatus = "⚠️ Copy failed"
		return
	}

	if cb.language != "" {
		m.reloadStatus = fmt.Sprintf("📋 Copied! (%s, %d lines)", cb.language, cb.lineCount)
	} else {
		m.reloadStatus = fmt.Sprintf("📋 Copied! (%d lines)", cb.lineCount)
	}
}

// decorateCodeBlockLine は行頭の余白（または先頭）にコピー選択マーカーを付与します
func decorateCodeBlockLine(line string, active bool) string {
	marker := "\x1b[38;5;244m│ \x1b[0m"
	if active {
		marker = "\x1b[48;5;214m\x1b[38;5;0m▌\x1b[0m "
	}
	if strings.HasPrefix(line, "  ") {
		return marker + line[2:]
	}
	return marker + line
}

// updateCopyHighlight はコピーモード中のコードブロック選択ハイライトを反映します
func (m *model) updateCopyHighlight(autoScroll bool) {
	if len(m.pristineLines) == 0 {
		return
	}

	if len(m.renderedLines) != len(m.pristineLines) {
		m.renderedLines = make([]string, len(m.pristineLines))
	}
	copy(m.renderedLines, m.pristineLines)

	if !m.copyMode || len(m.codeBlocks) == 0 {
		m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
		return
	}

	if m.currentCodeIndex < 0 {
		m.currentCodeIndex = 0
	}
	if m.currentCodeIndex >= len(m.codeBlocks) {
		m.currentCodeIndex = len(m.codeBlocks) - 1
	}

	for idx, cb := range m.codeBlocks {
		isActive := idx == m.currentCodeIndex
		for lineIdx := cb.startLine; lineIdx <= cb.endLine && lineIdx < len(m.renderedLines); lineIdx++ {
			if lineIdx < 0 {
				continue
			}
			m.renderedLines[lineIdx] = decorateCodeBlockLine(m.pristineLines[lineIdx], isActive)
		}
	}

	m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))

	if autoScroll {
		cb := m.codeBlocks[m.currentCodeIndex]
		top := m.viewport.YOffset
		bottom := top + m.viewport.Height - 1

		// 選択したコードブロックの先頭または末尾が画面外ならスクロールして見やすく表示する
		if cb.startLine < top || cb.startLine > bottom {
			yOffset := cb.startLine - 2
			if yOffset < 0 {
				yOffset = 0
			}
			maxOffset := len(m.renderedLines) - m.viewport.Height
			if maxOffset < 0 {
				maxOffset = 0
			}
			if yOffset > maxOffset {
				yOffset = maxOffset
			}
			m.viewport.SetYOffset(yOffset)
		}
	}
}
