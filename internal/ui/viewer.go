package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	"github.com/nobarudo/gomad/internal/sideber"
	"github.com/nobarudo/gomad/internal/watcher"
)

type model struct {
	filePath          string
	content           string
	isStdin           bool
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
	watcher           *watcher.Watcher
	reloadStatus      string
	lastReloadTime    time.Time
}
