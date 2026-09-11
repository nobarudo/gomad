package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
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
