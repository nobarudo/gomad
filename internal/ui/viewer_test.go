package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"gomad/internal/watcher"
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
