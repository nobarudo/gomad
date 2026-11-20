package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
)

func generateLargeMarkdown(sections int) string {
	var b strings.Builder
	b.WriteString("# Large Benchmark Document\n\n")
	for i := 0; i < sections; i++ {
		fmt.Fprintf(&b, "## Section %d\n\n", i+1)
		b.WriteString("This is a paragraph with **bold**, *italic*, `inline code`, and a [link](https://example.com/page) plus a [local link](README.md).\n\n")
		b.WriteString("```go\nfunc hello(name string) string {\n\treturn \"Hello, \" + name\n}\n```\n\n")
		b.WriteString("- First bullet item with keyword gomad\n- Second bullet item\n- Third bullet item\n\n")
	}
	return b.String()
}

func BenchmarkRenderContent_Cold(b *testing.B) {
	doc := generateLargeMarkdown(50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := model{
			width:        100,
			height:       40,
			ready:        true,
			content:      doc,
			currentStyle: "dark",
			viewport:     viewport.New(100, 38),
		}
		if err := m.renderContent(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderContent_ToggleSidebar(b *testing.B) {
	doc := generateLargeMarkdown(50)
	m := model{
		width:        100,
		height:       40,
		ready:        true,
		content:      doc,
		currentStyle: "dark",
		viewport:     viewport.New(100, 38),
	}
	if err := m.renderContent(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.showSidebar = !m.showSidebar
		m.calculateLayout()
		if err := m.renderContent(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteSearch(b *testing.B) {
	doc := generateLargeMarkdown(100)
	m := model{
		width:        100,
		height:       40,
		ready:        true,
		content:      doc,
		currentStyle: "dark",
		viewport:     viewport.New(100, 38),
	}
	if err := m.renderContent(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.executeSearch("gomad")
	}
}

func BenchmarkUpdateHighlight(b *testing.B) {
	doc := generateLargeMarkdown(100)
	m := model{
		width:        100,
		height:       40,
		ready:        true,
		content:      doc,
		currentStyle: "dark",
		viewport:     viewport.New(100, 38),
	}
	if err := m.renderContent(); err != nil {
		b.Fatal(err)
	}
	m.executeSearch("Section 50")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.updateHighlight()
	}
}

func BenchmarkStripANSI(b *testing.B) {
	line := "\x1b[38;5;63m\x1b[1mHello\x1b[0m World with \x1b[48;5;236m\x1b[38;5;81minline code\x1b[0m and plain text"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stripANSI(line)
	}
}
