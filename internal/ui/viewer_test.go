package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nobarudo/gomad/internal/watcher"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"\x1b[1mHello\x1b[0m World", "Hello World"},
		{"Plain text", "Plain text"},
		{"\x1b[38;5;63mStyled text\x1b[0m", "Styled text"},
	}

	for _, tc := range tests {
		got := stripANSI(tc.input)
		if got != tc.expected {
			t.Errorf("stripANSI(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestParseSegments(t *testing.T) {
	input := "\x1b[1mHi\x1b[0m"
	segs := parseSegments(input)
	if len(segs) != 4 {
		t.Fatalf("Expected 4 segments, got %d", len(segs))
	}
	if segs[0].text != "\x1b[1m" || !segs[0].isANSI {
		t.Errorf("Segment 0 incorrect: %+v", segs[0])
	}
	if segs[1].text != "H" || segs[1].isANSI {
		t.Errorf("Segment 1 incorrect: %+v", segs[1])
	}
	if segs[2].text != "i" || segs[2].isANSI {
		t.Errorf("Segment 2 incorrect: %+v", segs[2])
	}
	if segs[3].text != "\x1b[0m" || !segs[3].isANSI {
		t.Errorf("Segment 3 incorrect: %+v", segs[3])
	}
}

func TestHighlightQueryInLine(t *testing.T) {
	tests := []struct {
		input    string
		query    string
		expected string
	}{
		{"Go is great", "go", "<Go> is great"},
		{"\x1b[1mGo is great\x1b[0m", "is", "\x1b[1mGo <is> great\x1b[0m"},
		{"No match", "rust", "No match"},
		{"test test", "test", "<test> <test>"},
		{"日本語の見出し", "見出し", "日本語の<見出し>"},
		{"│ table match │", "match", "│ table <match> │"},
	}

	for _, tc := range tests {
		got := highlightQueryInLine(tc.input, tc.query, "<", ">")
		if got != tc.expected {
			t.Errorf("highlightQueryInLine(%q, %q) = %q, expected %q", tc.input, tc.query, got, tc.expected)
		}
	}
}

func TestExecuteSearch(t *testing.T) {
	m := model{
		pristineLines: []string{
			"Go is an open-source programming language",
			"Created by Google",
			"It is also known as Golang",
		},
		renderedLines: []string{
			"Go is an open-source programming language",
			"Created by Google",
			"It is also known as Golang",
		},
		viewport: viewport.New(80, 24),
	}

	// Test Case 1: Search term that matches multiple lines (case-insensitive)
	m.executeSearch("go")
	expectedResults1 := []int{0, 1, 2}
	if len(m.searchResults) != len(expectedResults1) {
		t.Fatalf("Expected %d results, got %d", len(expectedResults1), len(m.searchResults))
	}
	for i, v := range m.searchResults {
		if v != expectedResults1[i] {
			t.Errorf("Expected searchResults[%d] = %d, got %d", i, expectedResults1[i], v)
		}
	}
	if m.searchQuery != "go" {
		t.Errorf("Expected searchQuery to be 'go', got %q", m.searchQuery)
	}

	// Test Case 2: Search term that matches a single line
	m.executeSearch("google")
	expectedResults2 := []int{1}
	if len(m.searchResults) != len(expectedResults2) {
		t.Fatalf("Expected %d results, got %d", len(expectedResults2), len(m.searchResults))
	}
	if m.searchResults[0] != expectedResults2[0] {
		t.Errorf("Expected searchResults[0] = %d, got %d", expectedResults2[0], m.searchResults[0])
	}

	// Test Case 3: Search term with no matches
	m.executeSearch("rust")
	if len(m.searchResults) != 0 {
		t.Errorf("Expected 0 results for non-matching query, got %d", len(m.searchResults))
	}

	// Test Case 4: Empty search term resets highlights
	m.executeSearch("")
	if len(m.searchResults) != 0 {
		t.Errorf("Expected 0 results for empty query, got %d", len(m.searchResults))
	}
	if m.searchQuery != "" {
		t.Errorf("Expected empty searchQuery, got %q", m.searchQuery)
	}
}

func TestUpdateHighlight(t *testing.T) {
	m := model{
		pristineLines: []string{
			"Line 1",
			"Line 2",
			"Line 3",
		},
		viewport: viewport.New(80, 24),
	}

	m.searchResults = []int{0, 2}
	m.currentMatchIndex = 0

	m.updateHighlight()

	// The first line should contain the highlighted query "Line 1"
	if !strings.Contains(stripANSI(m.renderedLines[0]), "Line 1") {
		t.Errorf("Expected line 1 to contain 'Line 1', got %q", stripANSI(m.renderedLines[0]))
	}

	// The second line should NOT contain search matches for "Line 1" (since search is "Line 1")
	m.searchQuery = "Line"
	m.updateHighlight()

	if !strings.Contains(stripANSI(m.renderedLines[0]), "Line 1") {
		t.Errorf("Expected line 1 to contain 'Line 1', got %q", stripANSI(m.renderedLines[0]))
	}

	// Navigation to next match
	m.currentMatchIndex = 1
	m.updateHighlight()

	if !strings.Contains(stripANSI(m.renderedLines[2]), "Line 3") {
		t.Errorf("Expected line 3 to contain 'Line 3', got %q", stripANSI(m.renderedLines[2]))
	}
}

func TestSidebarToggleAndLayout(t *testing.T) {
	m := model{
		width:        100,
		height:       30,
		ready:        true,
		content:      "# Heading 1\n\nText 1\n\n## Heading 2\n\nText 2",
		currentStyle: "dark",
		viewport:     viewport.New(100, 28),
	}

	if err := m.renderContent(); err != nil {
		t.Fatalf("renderContent failed: %v", err)
	}

	if m.showSidebar {
		t.Errorf("Expected sidebar initially hidden")
	}

	// 't' キーで開く
	m.calculateLayout()
	if m.viewport.Width != 100 {
		t.Errorf("Expected viewport width 100, got %d", m.viewport.Width)
	}

	m.showSidebar = true
	m.calculateLayout()

	if m.viewport.Width >= 100 {
		t.Errorf("Expected viewport width to shrink when sidebar is open, got %d", m.viewport.Width)
	}

	if len(m.sidebar.Items()) != 2 {
		t.Fatalf("Expected 2 items in sidebar, got %d", len(m.sidebar.Items()))
	}
	if m.sidebar.Items()[0].Level != 1 {
		t.Errorf("Expected item 0 level 1, got %d", m.sidebar.Items()[0].Level)
	}
	if m.sidebar.Items()[1].Level != 2 {
		t.Errorf("Expected item 1 level 2, got %d", m.sidebar.Items()[1].Level)
	}

	// 閉じる
	m.showSidebar = false
	m.calculateLayout()
	if m.viewport.Width != 100 {
		t.Errorf("Expected viewport width 100 after closing sidebar, got %d", m.viewport.Width)
	}
}

func TestSidebarEnterKeepOpenAndTab(t *testing.T) {
	// 長いコンテンツにしてスクロール可能にする
	var longContent strings.Builder
	longContent.WriteString("# First\n\n")
	for i := 0; i < 20; i++ {
		longContent.WriteString("Some line of text\n\n")
	}
	longContent.WriteString("## Second\n\nMore text\n\n")
	for i := 0; i < 20; i++ {
		longContent.WriteString("Trailing line of text\n\n")
	}

	m := model{
		width:        100,
		height:       10,
		ready:        true,
		content:      longContent.String(),
		currentStyle: "dark",
		viewport:     viewport.New(100, 8),
	}

	if err := m.renderContent(); err != nil {
		t.Fatalf("renderContent failed: %v", err)
	}

	// 1. 't' キーを押して開く
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updatedM.(model)
	if !m.showSidebar {
		t.Fatalf("Expected showSidebar to be true")
	}
	if !m.sidebarFocused {
		t.Fatalf("Expected sidebarFocused to be true")
	}

	// 2. 'j' でカーソルを2番目の見出しへ
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updatedM.(model)
	if m.sidebar.Cursor() != 1 {
		t.Errorf("Expected sidebar cursor 1, got %d", m.sidebar.Cursor())
	}

	// 3. 'enter' を押してジャンプ -> サイドバーは開いたまま（showSidebar == true）
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedM.(model)
	if !m.showSidebar {
		t.Errorf("Expected showSidebar to remain true after Enter")
	}
	if m.viewport.YOffset != m.headingLines[1] {
		t.Errorf("Expected viewport offset %d, got %d", m.headingLines[1], m.viewport.YOffset)
	}

	// 4. 'tab' を押してフォーカス切り替え
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedM.(model)
	if m.sidebarFocused {
		t.Errorf("Expected sidebarFocused to be false after Tab")
	}
	if !m.showSidebar {
		t.Errorf("Expected showSidebar to remain true after Tab")
	}

	// 5. 'tab' をもう一度押してサイドバーフォーカス復帰
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updatedM.(model)
	if !m.sidebarFocused {
		t.Errorf("Expected sidebarFocused to be true after second Tab")
	}
}

func TestSidebarFocusSwitchingHL(t *testing.T) {
	m := model{
		width:        100,
		height:       20,
		ready:        true,
		content:      "# Heading 1\n\nContent",
		currentStyle: "dark",
		viewport:     viewport.New(100, 18),
	}

	if err := m.renderContent(); err != nil {
		t.Fatalf("renderContent failed: %v", err)
	}

	// 1. 't' キーで目次サイドバーを開く
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updatedM.(model)
	if !m.showSidebar || !m.sidebarFocused {
		t.Fatalf("Expected sidebar open and focused")
	}

	// 2. 'l' キーで本文（右）へフォーカス移動
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updatedM.(model)
	if m.sidebarFocused {
		t.Errorf("Expected sidebarFocused to be false after 'l'")
	}

	// 3. 本文にいる状態で 'l' を押しても右端なので本文のまま
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updatedM.(model)
	if m.sidebarFocused {
		t.Errorf("Expected sidebarFocused to remain false after redundant 'l'")
	}

	// 4. 'h' キーでサイドバー（左）へフォーカス移動
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updatedM.(model)
	if !m.sidebarFocused {
		t.Errorf("Expected sidebarFocused to be true after 'h'")
	}

	// 5. サイドバーにいる状態で 'h' を押しても左端なのでサイドバーのまま
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updatedM.(model)
	if !m.sidebarFocused {
		t.Errorf("Expected sidebarFocused to remain true after redundant 'h'")
	}

	// 6. 矢印キー 'right' で本文へ移動
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updatedM.(model)
	if m.sidebarFocused {
		t.Errorf("Expected sidebarFocused to be false after 'right'")
	}

	// 7. 矢印キー 'left' でサイドバーへ移動
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = updatedM.(model)
	if !m.sidebarFocused {
		t.Errorf("Expected sidebarFocused to be true after 'left'")
	}
}

func TestFileReloadMsg(t *testing.T) {
	initialMD := "# Heading 1\n\nInitial paragraph"
	m := model{
		width:        80,
		height:       24,
		ready:        true,
		content:      initialMD,
		currentStyle: "dark",
		viewport:     viewport.New(80, 22),
	}
	if err := m.renderContent(); err != nil {
		t.Fatalf("renderContent failed: %v", err)
	}

	if len(m.headingLines) != 1 {
		t.Fatalf("Expected 1 heading, got %d", len(m.headingLines))
	}

	// 変更イベントを受信したとする
	newMD := "# Heading 1\n\nInitial paragraph\n\n## Heading 2\n\nNew paragraph"
	reloadMsg := fileReloadMsg{
		event: watcher.Event{
			Path:    "dummy.md",
			Content: newMD,
			ModTime: time.Now(),
		},
	}

	updatedM, _ := m.Update(reloadMsg)
	m = updatedM.(model)

	if m.content != newMD {
		t.Errorf("Expected updated content, got %q", m.content)
	}
	if len(m.headingLines) != 2 {
		t.Errorf("Expected 2 headings after reload, got %d", len(m.headingLines))
	}
	if !strings.Contains(m.reloadStatus, "⚡") {
		t.Errorf("Expected reloadStatus to contain '⚡', got %q", m.reloadStatus)
	}
}

func TestManualReloadKey(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.md")

	if err := os.WriteFile(testFile, []byte("# Initial"), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	m := model{
		filePath:     testFile,
		width:        80,
		height:       24,
		ready:        true,
		content:      "# Initial",
		currentStyle: "dark",
		viewport:     viewport.New(80, 22),
	}
	if err := m.renderContent(); err != nil {
		t.Fatalf("renderContent failed: %v", err)
	}

	// ファイルの内容を更新
	if err := os.WriteFile(testFile, []byte("# Updated manually"), 0644); err != nil {
		t.Fatalf("Failed to write updated file: %v", err)
	}

	// 'r' キーを押す
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updatedM.(model)

	if !strings.Contains(m.content, "Updated manually") {
		t.Errorf("Expected content to contain 'Updated manually', got %q", m.content)
	}
	if !strings.Contains(m.reloadStatus, "⚡") {
		t.Errorf("Expected reloadStatus to contain '⚡', got %q", m.reloadStatus)
	}
}

func TestAutoReloadView(t *testing.T) {
	m := model{
		filePath:     "test.md",
		width:        80,
		height:       24,
		ready:        true,
		content:      "# Test",
		currentStyle: "dark",
		viewport:     viewport.New(80, 22),
		reloadStatus: "⚡ 12:34:56",
	}

	view := m.View()
	if !strings.Contains(view, "'r':リロード") {
		t.Errorf("Expected header to contain ''r':リロード', got: %s", view)
	}
	if !strings.Contains(view, "⚡ 12:34:56") {
		t.Errorf("Expected footer to contain '⚡ 12:34:56', got: %s", view)
	}
}

func TestStdinModelView(t *testing.T) {
	m := model{
		filePath:     "(stdin)",
		isStdin:      true,
		width:        80,
		height:       24,
		ready:        true,
		content:      "# Stdin Content",
		currentStyle: "dark",
		viewport:     viewport.New(80, 22),
	}

	view := m.View()
	if !strings.Contains(view, "(stdin)") {
		t.Errorf("Expected header to contain '(stdin)', got: %s", view)
	}
	if strings.Contains(view, "'r':リロード") {
		t.Errorf("Expected header NOT to contain ''r':リロード' for stdin, got: %s", view)
	}
}

func TestStdinManualReloadIgnored(t *testing.T) {
	m := model{
		filePath:     "(stdin)",
		isStdin:      true,
		width:        80,
		height:       24,
		ready:        true,
		content:      "# Stdin Content",
		currentStyle: "dark",
		viewport:     viewport.New(80, 22),
	}

	// 'r' キーを押しても何も変化しない
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updatedM.(model)

	if m.content != "# Stdin Content" {
		t.Errorf("Expected content to remain unchanged, got %q", m.content)
	}
	if m.reloadStatus != "" {
		t.Errorf("Expected reloadStatus to remain empty, got %q", m.reloadStatus)
	}
}

func TestSidebarSyncWithContentScroll(t *testing.T) {
	// 3つの見出しを持つドキュメント
	var doc strings.Builder
	doc.WriteString("# Section 1\n\n")
	for i := 0; i < 15; i++ {
		doc.WriteString("Paragraph text in section 1.\n\n")
	}
	doc.WriteString("## Section 2\n\n")
	for i := 0; i < 15; i++ {
		doc.WriteString("Paragraph text in section 2.\n\n")
	}
	doc.WriteString("### Section 3\n\n")
	for i := 0; i < 15; i++ {
		doc.WriteString("Paragraph text in section 3.\n\n")
	}

	m := model{
		width:        100,
		height:       15,
		ready:        true,
		content:      doc.String(),
		currentStyle: "dark",
		viewport:     viewport.New(100, 13),
	}

	if err := m.renderContent(); err != nil {
		t.Fatalf("renderContent failed: %v", err)
	}

	if len(m.headingLines) != 3 {
		t.Fatalf("Expected 3 heading lines, got %d", len(m.headingLines))
	}

	// 1. 't' キーでサイドバーを開く
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updatedM.(model)
	if m.sidebar.Cursor() != 0 {
		t.Errorf("Expected initial sidebar cursor 0, got %d", m.sidebar.Cursor())
	}

	// 2. 'l' キーで本文にフォーカスを移動
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updatedM.(model)
	if m.sidebarFocused {
		t.Fatalf("Expected focus in main content")
	}

	// 3. ']' (nextHeading) キーで Section 1 (line 1) へジャンプ
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = updatedM.(model)
	if m.sidebar.Cursor() != 0 {
		t.Errorf("Expected sidebar cursor 0 after jumping to section 1, got %d", m.sidebar.Cursor())
	}

	// 4. ']' キーで次の見出し (Section 2, line 33) へジャンプ
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = updatedM.(model)
	if m.sidebar.Cursor() != 1 {
		t.Errorf("Expected sidebar cursor 1 after jumping to section 2, got %d", m.sidebar.Cursor())
	}

	// 5. ']' キーでさらに次の見出し (Section 3, line 65) へジャンプ
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = updatedM.(model)
	if m.sidebar.Cursor() != 2 {
		t.Errorf("Expected sidebar cursor 2 after jumping to section 3, got %d", m.sidebar.Cursor())
	}

	// 6. 'G' キーで最下部に移動 -> Section 3 (cursor 2) に同期される
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updatedM.(model)
	if m.sidebar.Cursor() != 2 {
		t.Errorf("Expected sidebar cursor 2 after GotoBottom, got %d", m.sidebar.Cursor())
	}

	// 7. 'g' キーで先頭に戻る
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updatedM.(model)
	if m.sidebar.Cursor() != 0 {
		t.Errorf("Expected sidebar cursor 0 after goto top, got %d", m.sidebar.Cursor())
	}

	// 8. 'h' キーでサイドバーにフォーカスを戻し、手動でカーソルを動かす
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updatedM.(model)
	if !m.sidebarFocused {
		t.Fatalf("Expected focus in sidebar")
	}

	// サイドバーフォーカス時に 'j' でカーソルを動かす -> 本文スクロールによる上書きは発生しない
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updatedM.(model)
	if m.sidebar.Cursor() != 1 {
		t.Errorf("Expected sidebar cursor 1 after manual 'j', got %d", m.sidebar.Cursor())
	}

	// 9. サイドバーから 'l' で本文に戻ると、本文の現在位置 (先頭: line 0) に再同期される
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updatedM.(model)
	if m.sidebarFocused {
		t.Fatalf("Expected focus in main content")
	}
	if m.sidebar.Cursor() != 0 {
		t.Errorf("Expected sidebar cursor to resync to 0 after returning to content, got %d", m.sidebar.Cursor())
	}
}

func TestCyanThemeRenderAndPicker(t *testing.T) {
	doc := "# Title\n\n## Section 2\n\nSome paragraph with `code` and [link](https://example.com)"
	m := model{
		width:        80,
		height:       24,
		ready:        true,
		content:      doc,
		currentStyle: "cyan",
		viewport:     viewport.New(80, 22),
	}

	// 1. cyan スタイルで正常にレンダリングできること
	if err := m.renderContent(); err != nil {
		t.Fatalf("renderContent with cyan theme failed: %v", err)
	}

	// 2. pink レイアウト特有の見出しプレフィックス '▌' が含まれていること
	renderedText := stripANSI(m.viewport.View())
	if !strings.Contains(renderedText, "▌") {
		t.Errorf("Expected rendered markdown to contain '▌' from cyan theme, got:\n%s", renderedText)
	}

	// 3. テーマ選択モーダルを開いた時、availableStyles に cyan が含まれていること
	m.availableStyles = append([]string{"dark"}, "cyan")
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updatedM.(model)
	if !m.showStylePicker {
		t.Fatalf("Expected showStylePicker to be true after 's'")
	}
	pickerView := m.stylePickerView()
	if !strings.Contains(pickerView, "cyan") {
		t.Errorf("Expected style picker to display 'cyan', got:\n%s", pickerView)
	}
}

func TestResolveEditorCmd(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	bin, args := resolveEditorCmd("sample.md")
	if bin == "" {
		t.Fatalf("Expected fallback editor binary, got empty")
	}
	if len(args) != 1 || args[0] != "sample.md" {
		t.Errorf("Expected args [sample.md], got %v", args)
	}

	// EDITOR with flags
	t.Setenv("EDITOR", "code --wait")
	bin, args = resolveEditorCmd("sample.md")
	if bin != "code" {
		t.Errorf("Expected bin 'code', got %q", bin)
	}
	if len(args) != 2 || args[0] != "--wait" || args[1] != "sample.md" {
		t.Errorf("Expected args [--wait sample.md], got %v", args)
	}

	// VISUAL takes precedence over EDITOR
	t.Setenv("VISUAL", "nvim")
	bin, args = resolveEditorCmd("sample.md")
	if bin != "nvim" {
		t.Errorf("Expected VISUAL 'nvim' to take precedence, got %q", bin)
	}
	if len(args) != 1 || args[0] != "sample.md" {
		t.Errorf("Expected args [sample.md], got %v", args)
	}
}

func TestOpenEditorKeyAndFinishedMsg(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.md")

	if err := os.WriteFile(testFile, []byte("# Before Edit"), 0644); err != nil {
		t.Fatalf("Failed to write initial file: %v", err)
	}

	m := model{
		filePath:     testFile,
		width:        80,
		height:       24,
		ready:        true,
		content:      "# Before Edit",
		currentStyle: "dark",
		viewport:     viewport.New(80, 22),
	}
	if err := m.renderContent(); err != nil {
		t.Fatalf("renderContent failed: %v", err)
	}

	// 1. 'e' キーを押すと外部エディタ起動の tea.Cmd が返る
	updatedM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updatedM.(model)
	if cmd == nil {
		t.Fatalf("Expected non-nil tea.Cmd when pressing 'e' on a file")
	}

	// 2. エディタでファイルが書き換えられて終了したと想定
	if err := os.WriteFile(testFile, []byte("# After Edit"), 0644); err != nil {
		t.Fatalf("Failed to write updated file: %v", err)
	}

	updatedM, _ = m.Update(editorFinishedMsg{err: nil})
	m = updatedM.(model)

	if !strings.Contains(m.content, "After Edit") {
		t.Errorf("Expected content to be reloaded with 'After Edit', got %q", m.content)
	}
	if !strings.Contains(m.reloadStatus, "⚡") {
		t.Errorf("Expected reloadStatus to contain '⚡', got %q", m.reloadStatus)
	}

	// 3. Stdin モードでは 'e' キーは無視される
	stdinModel := model{
		filePath:     "(stdin)",
		isStdin:      true,
		width:        80,
		height:       24,
		ready:        true,
		content:      "# Stdin",
		currentStyle: "dark",
		viewport:     viewport.New(80, 22),
	}
	_, stdinCmd := stdinModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if stdinCmd != nil {
		t.Errorf("Expected nil cmd when pressing 'e' in stdin mode")
	}
}

func TestLinkFocusCycleAndLocalFileNavigation(t *testing.T) {
	tempDir := t.TempDir()
	firstFile := filepath.Join(tempDir, "first.md")
	secondFile := filepath.Join(tempDir, "second.md")

	firstMD := "# First Doc\n\nSee [Second Doc](second.md) or [Example](https://example.com) and `inline [ignored](ignore.md)`.\n\n```md\n[codeblock](ignored.md)\n```\n"
	secondMD := "# Second Doc\n\nWelcome to the second document!"

	if err := os.WriteFile(firstFile, []byte(firstMD), 0644); err != nil {
		t.Fatalf("Failed to write first file: %v", err)
	}
	if err := os.WriteFile(secondFile, []byte(secondMD), 0644); err != nil {
		t.Fatalf("Failed to write second file: %v", err)
	}

	m := model{
		filePath:     firstFile,
		width:        80,
		height:       24,
		ready:        true,
		content:      firstMD,
		currentStyle: "dark",
		viewport:     viewport.New(80, 22),
	}
	if err := m.renderContent(); err != nil {
		t.Fatalf("renderContent failed: %v", err)
	}

	// コードブロックとインラインコード内のリンクは除外され、2件だけ抽出されること
	if len(m.links) != 2 {
		t.Fatalf("Expected 2 links, got %d: %+v", len(m.links), m.links)
	}
	if m.links[0].url != "second.md" {
		t.Errorf("Expected link 0 url 'second.md', got %q", m.links[0].url)
	}
	if m.links[1].url != "https://example.com" {
		t.Errorf("Expected link 1 url 'https://example.com', got %q", m.links[1].url)
	}

	// 1. 初回の 'o' キーで画面内の最初のリンク (index 0) がハイライトされる
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = updatedM.(model)
	if !m.linkFocusActive {
		t.Fatalf("Expected linkFocusActive to be true after pressing 'o'")
	}
	if m.currentLinkIndex != 0 {
		t.Errorf("Expected currentLinkIndex 0, got %d", m.currentLinkIndex)
	}
	if !strings.Contains(m.View(), "🔗 (1/2) → second.md") {
		t.Errorf("Expected footer to show active link 1/2, got:\n%s", m.View())
	}

	// 2. もう一度 'o' キーを押すと次のリンク (index 1) へ移動する
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = updatedM.(model)
	if m.currentLinkIndex != 1 {
		t.Errorf("Expected currentLinkIndex 1, got %d", m.currentLinkIndex)
	}
	if !strings.Contains(m.View(), "🔗 (2/2) → https://example.com") {
		t.Errorf("Expected footer to show active link 2/2, got:\n%s", m.View())
	}

	// 3. 'O' キーで前のリンク (index 0) に戻る
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'O'}})
	m = updatedM.(model)
	if m.currentLinkIndex != 0 {
		t.Errorf("Expected currentLinkIndex 0 after 'O', got %d", m.currentLinkIndex)
	}

	// 4. 'Enter' キーでローカルの second.md へ遷移する
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedM.(model)
	if m.filePath != secondFile {
		t.Errorf("Expected filePath %q, got %q", secondFile, m.filePath)
	}
	if !strings.Contains(m.content, "Welcome to the second document!") {
		t.Errorf("Expected second document content, got %q", m.content)
	}
	if len(m.history) != 1 {
		t.Fatalf("Expected 1 history item, got %d", len(m.history))
	}

	// 5. 'Backspace' キーで元の first.md に戻る
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updatedM.(model)
	if m.filePath != firstFile {
		t.Errorf("Expected to navigate back to %q, got %q", firstFile, m.filePath)
	}
	if len(m.history) != 0 {
		t.Errorf("Expected history to be empty after navigating back, got %d", len(m.history))
	}
}

func TestFindMarkdownFilesAndDirectoryPicker(t *testing.T) {
	tempDir := t.TempDir()

	// 空のディレクトリはエラーになること
	if _, _, err := findMarkdownFiles(tempDir); err == nil {
		t.Fatalf("Expected error when no markdown files exist in directory")
	}

	alphaFile := filepath.Join(tempDir, "alpha.md")
	readmeFile := filepath.Join(tempDir, "README.md")
	zetaFile := filepath.Join(tempDir, "zeta.md")
	nonMdFile := filepath.Join(tempDir, "notes.txt")

	_ = os.WriteFile(alphaFile, []byte("# Alpha"), 0644)
	_ = os.WriteFile(readmeFile, []byte("# Project README"), 0644)
	_ = os.WriteFile(zetaFile, []byte("# Zeta"), 0644)
	_ = os.WriteFile(nonMdFile, []byte("plain text"), 0644)

	files, readmeIdx, err := findMarkdownFiles(tempDir)
	if err != nil {
		t.Fatalf("findMarkdownFiles failed: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("Expected 3 markdown files, got %d: %v", len(files), files)
	}
	// アルファベット順で alpha.md (0), README.md (1), zeta.md (2) となり、readmeIdx は 1 になること
	if readmeIdx != 1 || filepath.Base(files[readmeIdx]) != "README.md" {
		t.Errorf("Expected readmeIdx 1 (README.md), got %d (%v)", readmeIdx, files)
	}

	// ディレクトリ指定起動状態のモデルを構築
	m := model{
		filePath:       files[readmeIdx],
		content:        "# Project README",
		width:          80,
		height:         24,
		ready:          true,
		currentStyle:   "dark",
		viewport:       viewport.New(80, 22),
		dirPath:        tempDir,
		dirFiles:       files,
		fileIndex:      readmeIdx,
		showFilePicker: true,
	}
	if err := m.renderContent(); err != nil {
		t.Fatalf("renderContent failed: %v", err)
	}

	// 1. 起動直後はファイル一覧モーダルが表示され、README.md が初期選択されていること
	view := m.View()
	if !strings.Contains(view, "Select Markdown File") {
		t.Errorf("Expected file picker modal in view, got:\n%s", view)
	}
	if !strings.Contains(view, "README.md (active)") {
		t.Errorf("Expected README.md (active) in file picker modal, got:\n%s", view)
	}

	// 2. 'k' で alpha.md (index 0) を選択し、'Enter' で開く
	updatedM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updatedM.(model)
	if m.fileIndex != 0 {
		t.Errorf("Expected fileIndex 0 after 'k', got %d", m.fileIndex)
	}

	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updatedM.(model)
	if m.showFilePicker {
		t.Errorf("Expected showFilePicker to be false after Enter")
	}
	if m.filePath != alphaFile {
		t.Errorf("Expected filePath %q, got %q", alphaFile, m.filePath)
	}
	if !strings.Contains(m.content, "# Alpha") {
		t.Errorf("Expected content '# Alpha', got %q", m.content)
	}

	// 3. 'p' キーで再度ファイル一覧モーダルを開けること
	updatedM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updatedM.(model)
	if !m.showFilePicker {
		t.Errorf("Expected showFilePicker to be true after pressing 'p'")
	}

	// 4. 単一ファイル指定で開いた状態からでも 'p' キーで同ディレクトリ内のファイル一覧を開いて移動できること
	singleFileModel := model{
		filePath:     zetaFile,
		content:      "# Zeta",
		width:        80,
		height:       24,
		ready:        true,
		currentStyle: "dark",
		viewport:     viewport.New(80, 22),
	}
	if err := singleFileModel.renderContent(); err != nil {
		t.Fatalf("renderContent failed: %v", err)
	}

	updatedM, _ = singleFileModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	singleFileModel = updatedM.(model)
	if !singleFileModel.showFilePicker {
		t.Fatalf("Expected showFilePicker to be true when pressing 'p' in single-file mode")
	}
	if len(singleFileModel.dirFiles) != 3 {
		t.Fatalf("Expected 3 files in dirFiles, got %d", len(singleFileModel.dirFiles))
	}
	if singleFileModel.fileIndex != 2 {
		t.Errorf("Expected cursor on zeta.md (index 2), got %d", singleFileModel.fileIndex)
	}

	// 'k' で README.md (index 1) に移動して Enter で開く
	updatedM, _ = singleFileModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	singleFileModel = updatedM.(model)
	updatedM, _ = singleFileModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	singleFileModel = updatedM.(model)
	if singleFileModel.filePath != readmeFile {
		t.Errorf("Expected switched file to be %q, got %q", readmeFile, singleFileModel.filePath)
	}
}
