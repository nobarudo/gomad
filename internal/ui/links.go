package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nobarudo/gomad/internal/watcher"
)

type linkItem struct {
	text      string
	url       string
	line      int
	startRune int
	endRune   int
}

type historyItem struct {
	filePath string
	content  string
	yOffset  int
}

type browserOpenedMsg struct {
	url string
	err error
}

var (
	// Markdownリンク [text](url) または自動リンク <https://...> / 裸のURL
	mdLinkRegex   = regexp.MustCompile(`(!?)\[([^\]]+)\]\(([^)]+)\)|<(https?://[^>]+)>`)
	inlineCodeReg = regexp.MustCompile("`[^`]*`")
)

type rawLink struct {
	text string
	url  string
}

// extractRawLinks はMarkdown本文からコードブロックやインラインコードを除いてリンクを抽出します
func extractRawLinks(content string) []rawLink {
	lines := strings.Split(content, "\n")
	var rawLinks []rawLink
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// インラインコード内の [text](url) を除外するため空白に置換
		cleanedLine := inlineCodeReg.ReplaceAllStringFunc(line, func(s string) string {
			return strings.Repeat(" ", len(s))
		})

		matches := mdLinkRegex.FindAllStringSubmatch(cleanedLine, -1)
		for _, match := range matches {
			// 画像 ![alt](src) はスキップ
			if match[1] == "!" {
				continue
			}
			if match[4] != "" {
				// <https://...> 形式
				rawLinks = append(rawLinks, rawLink{
					text: match[4],
					url:  match[4],
				})
				continue
			}

			text := strings.TrimSpace(match[2])
			target := strings.TrimSpace(match[3])
			// [text](url "title") のタイトル部分を除去
			if idx := strings.IndexAny(target, " \t"); idx != -1 {
				target = strings.TrimSpace(target[:idx])
			}
			if target != "" {
				rawLinks = append(rawLinks, rawLink{
					text: text,
					url:  target,
				})
			}
		}
	}

	return rawLinks
}

func cleanMarkdownInline(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune("`*_~", r) {
			return -1
		}
		return r
	}, s)
}

// parseLinks はMarkdownテキストからリンクを抽出し、レンダリング後の行・文字位置にマッピングします
func parseLinks(content string, pristineLines []string) []linkItem {
	rawLinks := extractRawLinks(content)
	if len(rawLinks) == 0 || len(pristineLines) == 0 {
		return nil
	}

	plainLines := make([][]rune, len(pristineLines))
	for i, line := range pristineLines {
		plainLines[i] = []rune(stripANSI(line))
	}

	var items []linkItem
	searchLine := 0
	searchCol := 0

	for _, rl := range rawLinks {
		cleanText := cleanMarkdownInline(rl.text)
		candidates := []string{}
		if cleanText != "" && rl.url != "" && cleanText != rl.url {
			candidates = append(candidates, cleanText+" "+rl.url)
		}
		if cleanText != "" {
			candidates = append(candidates, cleanText)
		}
		if rl.url != "" {
			candidates = append(candidates, rl.url)
		}

		found := false
		for lineIdx := searchLine; lineIdx < len(plainLines); lineIdx++ {
			lineRunes := plainLines[lineIdx]
			startCol := 0
			if lineIdx == searchLine {
				startCol = searchCol
			}
			if startCol >= len(lineRunes) {
				continue
			}

			subLine := string(lineRunes[startCol:])
			for _, cand := range candidates {
				if cand == "" {
					continue
				}
				byteIdx := strings.Index(subLine, cand)
				if byteIdx != -1 {
					runeOffset := len([]rune(subLine[:byteIdx]))
					candRuneLen := len([]rune(cand))
					sRune := startCol + runeOffset
					eRune := sRune + candRuneLen

					// 同じ行の直後に URL が続いている場合はまとめてハイライト範囲に含める
					if cand == cleanText && rl.url != "" && cleanText != rl.url {
						rest := string(lineRunes[eRune:])
						if strings.HasPrefix(rest, " "+rl.url) {
							eRune += 1 + len([]rune(rl.url))
						}
					}

					items = append(items, linkItem{
						text:      rl.text,
						url:       rl.url,
						line:      lineIdx,
						startRune: sRune,
						endRune:   eRune,
					})
					searchLine = lineIdx
					searchCol = eRune
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}

	return items
}

// updateLinks はドキュメントのリンク一覧を再解析します
func (m *model) updateLinks() {
	m.links = parseLinks(m.content, m.pristineLines)
	if len(m.links) == 0 {
		m.linkFocusActive = false
		m.currentLinkIndex = 0
		return
	}
	if m.currentLinkIndex >= len(m.links) {
		m.currentLinkIndex = len(m.links) - 1
	}
	if m.linkFocusActive {
		m.updateLinkHighlight(false)
	}
}

// focusNextLink は 'o' キーで画面内（または次）のリンクにフォーカスを当て、押すごとに次のリンクへ移動します
func (m *model) focusNextLink() {
	if len(m.links) == 0 {
		m.linkFocusActive = false
		m.reloadStatus = "🔗 No links"
		return
	}

	// 検索ハイライトと競合しないようクリア
	if m.searchQuery != "" {
		m.searchQuery = ""
		m.searchResults = nil
		m.currentMatchIndex = 0
	}

	if !m.linkFocusActive {
		m.linkFocusActive = true
		// 現在表示中の画面内（YOffset以降）にある最初のリンクを探す
		idx := 0
		for i, link := range m.links {
			if link.line >= m.viewport.YOffset {
				idx = i
				break
			}
		}
		m.currentLinkIndex = idx
	} else {
		m.currentLinkIndex = (m.currentLinkIndex + 1) % len(m.links)
	}

	m.updateLinkHighlight(true)
}

// focusPrevLink は 'O' キーで前のリンクへフォーカスを移動します
func (m *model) focusPrevLink() {
	if len(m.links) == 0 {
		m.linkFocusActive = false
		m.reloadStatus = "🔗 No links"
		return
	}

	if m.searchQuery != "" {
		m.searchQuery = ""
		m.searchResults = nil
		m.currentMatchIndex = 0
	}

	if !m.linkFocusActive {
		m.linkFocusActive = true
		idx := len(m.links) - 1
		bottom := m.viewport.YOffset + m.viewport.Height - 1
		for i := len(m.links) - 1; i >= 0; i-- {
			if m.links[i].line <= bottom {
				idx = i
				break
			}
		}
		m.currentLinkIndex = idx
	} else {
		m.currentLinkIndex = (m.currentLinkIndex - 1 + len(m.links)) % len(m.links)
	}

	m.updateLinkHighlight(true)
}

// clearLinkFocus はリンクのハイライト選択状態を解除します
func (m *model) clearLinkFocus() {
	if !m.linkFocusActive {
		return
	}
	m.linkFocusActive = false
	if len(m.pristineLines) > 0 {
		m.renderedLines = make([]string, len(m.pristineLines))
		copy(m.renderedLines, m.pristineLines)
		m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
	}
}

// updateLinkHighlight はリンク箇所をハイライトしてビューポートを更新します
func (m *model) updateLinkHighlight(autoScroll bool) {
	if len(m.pristineLines) == 0 {
		return
	}

	m.renderedLines = make([]string, len(m.pristineLines))
	copy(m.renderedLines, m.pristineLines)

	if !m.linkFocusActive || len(m.links) == 0 {
		m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
		return
	}

	if m.currentLinkIndex < 0 {
		m.currentLinkIndex = 0
	}
	if m.currentLinkIndex >= len(m.links) {
		m.currentLinkIndex = len(m.links) - 1
	}

	// 行ごとにリンクをグループ化してハイライトを適用
	lineLinks := make(map[int][]int)
	for idx, link := range m.links {
		lineLinks[link.line] = append(lineLinks[link.line], idx)
	}

	for lineIdx, linkIndices := range lineLinks {
		if lineIdx < 0 || lineIdx >= len(m.renderedLines) {
			continue
		}
		m.renderedLines[lineIdx] = highlightLinkRangesInLine(
			m.pristineLines[lineIdx],
			linkIndices,
			m.links,
			m.currentLinkIndex,
		)
	}

	m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))

	if autoScroll {
		targetLine := m.links[m.currentLinkIndex].line
		// 画面外にある場合のみスクロールし、画面内にある場合は現在の表示位置を維持する
		if targetLine < m.viewport.YOffset || targetLine >= m.viewport.YOffset+m.viewport.Height {
			yOffset := targetLine - (m.viewport.Height / 2)
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

func highlightLinkRangesInLine(ansiStr string, linkIndices []int, allLinks []linkItem, activeIdx int) string {
	segments := parseSegments(ansiStr)
	var plainToSeg []int
	for segIdx, seg := range segments {
		if !seg.isANSI {
			plainToSeg = append(plainToSeg, segIdx)
		}
	}

	for _, idx := range linkIndices {
		link := allLinks[idx]
		if link.startRune < 0 || link.endRune > len(plainToSeg) || link.startRune >= link.endRune {
			continue
		}
		segStart := plainToSeg[link.startRune]
		segEnd := plainToSeg[link.endRune-1]

		startCode := "\x1b[48;5;229m\x1b[38;5;0m" // 非アクティブリンク: 柔らかい黄色背景
		if idx == activeIdx {
			startCode = "\x1b[48;5;214m\x1b[38;5;0m" // フォーカス中リンク: オレンジ背景
		}
		endCode := "\x1b[39;49m"

		segments[segStart].text = startCode + segments[segStart].text
		segments[segEnd].text = segments[segEnd].text + endCode
	}

	var result strings.Builder
	for _, seg := range segments {
		result.WriteString(seg.text)
	}
	return result.String()
}

// openSelectedLink は現在フォーカスされているリンクを開きます（ブラウザ起動・アンカージャンプ・ローカルファイル遷移）
func (m *model) openSelectedLink() tea.Cmd {
	if !m.linkFocusActive || len(m.links) == 0 || m.currentLinkIndex < 0 || m.currentLinkIndex >= len(m.links) {
		return nil
	}

	target := m.links[m.currentLinkIndex].url

	// 1. 外部URL (http / https / mailto) はシステムのデフォルトブラウザで開く
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:") {
		return openInBrowserCmd(target)
	}

	// 2. ページ内アンカー (#heading)
	if strings.HasPrefix(target, "#") {
		m.jumpToAnchor(strings.TrimPrefix(target, "#"))
		return nil
	}

	// 3. ローカルファイルへの遷移（#anchor 付きにも対応）
	filePart := target
	anchorPart := ""
	if hashIdx := strings.Index(target, "#"); hashIdx != -1 {
		filePart = target[:hashIdx]
		anchorPart = target[hashIdx+1:]
	}

	resolvedPath := filePart
	if !m.isStdin && m.filePath != "" && !filepath.IsAbs(filePart) {
		resolvedPath = filepath.Join(filepath.Dir(m.filePath), filePart)
	}
	resolvedPath = filepath.Clean(resolvedPath)

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		m.reloadStatus = fmt.Sprintf("⚠️ Cannot open: %s", filePart)
		return nil
	}

	// 現在のファイルを履歴スタックに保存
	m.history = append(m.history, historyItem{
		filePath: m.filePath,
		content:  m.content,
		yOffset:  m.viewport.YOffset,
	})

	m.filePath = resolvedPath
	m.content = string(data)
	m.isStdin = false
	m.linkFocusActive = false
	m.currentLinkIndex = 0
	m.searchQuery = ""
	m.searchResults = nil

	if err := m.renderContent(); err != nil {
		m.err = err
		return tea.Quit
	}

	if anchorPart != "" {
		m.jumpToAnchor(anchorPart)
	} else {
		m.viewport.GotoTop()
	}

	if m.showSidebar && !m.sidebarFocused {
		m.syncSidebarCursor()
	}

	// ファイル監視が有効な場合は遷移先ファイルに切り替え
	if m.watcher != nil {
		_ = m.watcher.Close()
		if newW, wErr := watcher.New(resolvedPath); wErr == nil {
			m.watcher = newW
			m.reloadStatus = "⚡ Auto-reload"
			return m.waitForFileChange()
		}
		m.watcher = nil
	}

	m.reloadStatus = fmt.Sprintf("📂 %s", filepath.Base(resolvedPath))
	return nil
}

// navigateBack はローカルファイル遷移の履歴から1つ前のファイル・スクロール位置に戻ります
func (m *model) navigateBack() tea.Cmd {
	if len(m.history) == 0 {
		return nil
	}

	prev := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]

	m.filePath = prev.filePath
	m.content = prev.content
	m.linkFocusActive = false
	m.currentLinkIndex = 0
	m.searchQuery = ""
	m.searchResults = nil

	// ファイルが存在する場合は最新の内容を読み直す
	if !m.isStdin && m.filePath != "" && m.filePath != "(stdin)" {
		if data, err := os.ReadFile(m.filePath); err == nil {
			m.content = string(data)
		}
	}

	if err := m.renderContent(); err != nil {
		m.err = err
		return tea.Quit
	}

	m.viewport.SetYOffset(prev.yOffset)
	if m.showSidebar && !m.sidebarFocused {
		m.syncSidebarCursor()
	}

	if m.watcher != nil && m.filePath != "" && m.filePath != "(stdin)" {
		_ = m.watcher.Close()
		if newW, wErr := watcher.New(m.filePath); wErr == nil {
			m.watcher = newW
			m.reloadStatus = "⚡ Auto-reload"
			return m.waitForFileChange()
		}
		m.watcher = nil
	}

	m.reloadStatus = fmt.Sprintf("↩️ %s", filepath.Base(m.filePath))
	return nil
}

func (m *model) jumpToAnchor(anchor string) {
	slug := normalizeSlug(anchor)
	for _, item := range m.sidebar.Items() {
		if normalizeSlug(item.Title) == slug || strings.Contains(normalizeSlug(item.Title), slug) {
			m.viewport.SetYOffset(item.Line)
			if m.showSidebar && !m.sidebarFocused {
				m.syncSidebarCursor()
			}
			return
		}
	}
}

func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r > 127 {
			return r
		}
		return -1
	}, s)
}

func openInBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		err := cmd.Start()
		return browserOpenedMsg{url: url, err: err}
	}
}
