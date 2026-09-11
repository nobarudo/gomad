package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	"gomad/internal/sideber"
)

type model struct {
	filePath          string
	content           string
	viewport          viewport.Model
	ready             bool
	err               error
	width             int
	height            int
	currentStyle      string
	showStylePicker   bool
	showHelpModal     bool
	showSidebar       bool
	sidebarFocused    bool
	sidebar           sideber.Model
	styleIndex        int
	availableStyles   []string
	renderedLines     []string
	headingLines      []int
	searchInput       textinput.Model
	showSearchInput   bool
	searchQuery       string
	searchResults     []int
	currentMatchIndex int
	pristineLines     []string
}
