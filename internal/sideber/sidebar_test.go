package sideber

import (
	"strings"
	"testing"
)

func TestParseItems(t *testing.T) {
	content := `# Document Title
Some introduction.

## Section 1
Content of section 1.

### Subsection 1.1
Detail here.

` + "```go\n# this is a comment, not a heading\nfunc main() {}\n```" + `

## Section 2
Final section.
`

	renderedLines := []string{
		"Document Title",
		"Some introduction.",
		"",
		"Section 1",
		"Content of section 1.",
		"",
		"Subsection 1.1",
		"Detail here.",
		"",
		"# this is a comment, not a heading",
		"",
		"Section 2",
		"Final section.",
	}

	items := ParseItems(content, renderedLines)
	if len(items) != 4 {
		t.Fatalf("Expected 4 items, got %d", len(items))
	}

	expected := []struct {
		level int
		title string
		line  int
	}{
		{1, "Document Title", 0},
		{2, "Section 1", 3},
		{3, "Subsection 1.1", 6},
		{2, "Section 2", 11},
	}

	for i, exp := range expected {
		if items[i].Level != exp.level {
			t.Errorf("Item %d: expected level %d, got %d", i, exp.level, items[i].Level)
		}
		if items[i].Title != exp.title {
			t.Errorf("Item %d: expected title %q, got %q", i, exp.title, items[i].Title)
		}
		if items[i].Line != exp.line {
			t.Errorf("Item %d: expected line %d, got %d", i, exp.line, items[i].Line)
		}
	}
}

func TestModelNavigation(t *testing.T) {
	m := New()
	items := []Item{
		{Level: 1, Title: "H1", Line: 0},
		{Level: 2, Title: "H2-1", Line: 5},
		{Level: 3, Title: "H3", Line: 10},
		{Level: 2, Title: "H2-2", Line: 15},
	}
	m.SetItems(items)
	m.SetSize(30, 20)

	if m.Cursor() != 0 {
		t.Errorf("Expected initial cursor 0, got %d", m.Cursor())
	}

	m.MoveDown()
	if m.Cursor() != 1 {
		t.Errorf("Expected cursor 1 after MoveDown, got %d", m.Cursor())
	}

	m.GotoBottom()
	if m.Cursor() != 3 {
		t.Errorf("Expected cursor 3 after GotoBottom, got %d", m.Cursor())
	}

	m.MoveDown() // should stay at bottom
	if m.Cursor() != 3 {
		t.Errorf("Expected cursor 3 at bottom limit, got %d", m.Cursor())
	}

	m.MoveUp()
	if m.Cursor() != 2 {
		t.Errorf("Expected cursor 2 after MoveUp, got %d", m.Cursor())
	}

	m.GotoTop()
	if m.Cursor() != 0 {
		t.Errorf("Expected cursor 0 after GotoTop, got %d", m.Cursor())
	}
}

func TestIndentInView(t *testing.T) {
	m := New()
	items := []Item{
		{Level: 1, Title: "Parent", Line: 0},
		{Level: 2, Title: "Child", Line: 5},
		{Level: 3, Title: "Grandchild", Line: 10},
	}
	m.SetItems(items)
	m.SetSize(30, 10)

	view := m.View()

	// レベル1はインデントなし: "# Parent"
	if !strings.Contains(view, "# Parent") {
		t.Errorf("Expected view to contain '# Parent', got:\n%s", view)
	}

	// レベル2は2スペースインデント: "  ## Child"
	if !strings.Contains(view, "  ## Child") {
		t.Errorf("Expected view to contain '  ## Child', got:\n%s", view)
	}

	// レベル3は4スペースインデント: "    ### Grandchild"
	if !strings.Contains(view, "    ### Grandchild") {
		t.Errorf("Expected view to contain '    ### Grandchild', got:\n%s", view)
	}
}

func TestParseItemsWithANSI(t *testing.T) {
	content := "### 環境変数ファイル(.env)の露出スキャン\n"
	renderedLines := []string{
		"\x1b[0m\x1b[38;5;39;1m  ### \x1b[0m\x1b[38;5;39;1m環境変数ファイル\x1b[0m\x1b[38;5;39;1m(.env)の露出スキャン\x1b[0m",
	}

	items := ParseItems(content, renderedLines)
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	if items[0].Line != 0 {
		t.Errorf("Expected line 0, got %d", items[0].Line)
	}
}
