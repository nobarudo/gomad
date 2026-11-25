package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"

	"github.com/nobarudo/gomad/internal/sideber"
	"github.com/nobarudo/gomad/themes"
)

type rendererCacheKey struct {
	style string
	width int
}

type cachedRender struct {
	rendered      string
	pristineLines []string
	plainLines    []string
	sidebarItems  []sideber.Item
	headingLines  []int
	links         []linkItem
	codeBlocks    []codeBlock
}

var (
	rendererCacheMu sync.Mutex
	rendererCache   = make(map[rendererCacheKey]*glamour.TermRenderer)
)

func getGlamourRenderer(style string, width int) (*glamour.TermRenderer, error) {
	key := rendererCacheKey{style: style, width: width}

	rendererCacheMu.Lock()
	if r, ok := rendererCache[key]; ok {
		rendererCacheMu.Unlock()
		return r, nil
	}
	rendererCacheMu.Unlock()

	var styleOpt glamour.TermRendererOption
	if jsonBytes, ok := themes.GetThemeJSON(style); ok {
		styleOpt = glamour.WithStylesFromJSONBytes(jsonBytes)
	} else {
		styleOpt = glamour.WithStandardStyle(style)
	}

	renderer, err := glamour.NewTermRenderer(
		styleOpt,
		glamour.WithWordWrap(width),
	)
	if err != nil {
		renderer, err = glamour.NewTermRenderer(
			glamour.WithStandardStyle("dark"),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create glamour renderer: %w", err)
		}
	}

	rendererCacheMu.Lock()
	if len(rendererCache) >= 32 {
		clear(rendererCache)
	}
	rendererCache[key] = renderer
	rendererCacheMu.Unlock()

	return renderer, nil
}

// Markdownコンテンツを指定スタイルでGlamourレンダリングする
func (m *model) renderContent() error {
	if m.renderCache == nil || m.renderCacheContent != m.content {
		m.renderCache = make(map[rendererCacheKey]*cachedRender)
		m.renderCacheContent = m.content
	}

	key := rendererCacheKey{style: m.currentStyle, width: m.viewport.Width}
	entry, hit := m.renderCache[key]

	if !hit {
		renderer, err := getGlamourRenderer(m.currentStyle, m.viewport.Width)
		if err != nil {
			return err
		}

		rendered, err := renderer.Render(m.content)
		if err != nil {
			return fmt.Errorf("failed to render markdown: %w", err)
		}

		pristineLines := strings.Split(rendered, "\n")
		plainLines := make([]string, len(pristineLines))
		for i, line := range pristineLines {
			plainLines[i] = stripANSI(line)
		}

		items := sideber.ParseItemsWithPlainLines(m.content, plainLines)
		hLines := make([]int, len(items))
		for i, item := range items {
			hLines[i] = item.Line
		}

		links := parseLinksWithPlainLines(m.content, plainLines)
		codeBlocks := parseCodeBlocksWithPlainLines(m.content, plainLines)

		entry = &cachedRender{
			rendered:      rendered,
			pristineLines: pristineLines,
			plainLines:    plainLines,
			sidebarItems:  items,
			headingLines:  hLines,
			links:         links,
			codeBlocks:    codeBlocks,
		}
		if len(m.renderCache) >= 8 {
			clear(m.renderCache)
		}
		m.renderCache[key] = entry
	}

	m.pristineLines = entry.pristineLines
	m.plainLines = entry.plainLines
	m.renderedLines = make([]string, len(entry.pristineLines))
	copy(m.renderedLines, entry.pristineLines)
	m.sidebar.SetItems(entry.sidebarItems)
	m.headingLines = entry.headingLines
	m.links = entry.links
	m.codeBlocks = entry.codeBlocks

	if len(m.links) == 0 {
		m.linkFocusActive = false
		m.currentLinkIndex = 0
	} else if m.currentLinkIndex >= len(m.links) {
		m.currentLinkIndex = len(m.links) - 1
	}

	if len(m.codeBlocks) == 0 {
		m.copyMode = false
		m.currentCodeIndex = 0
	} else if m.currentCodeIndex >= len(m.codeBlocks) {
		m.currentCodeIndex = len(m.codeBlocks) - 1
	}

	if m.copyMode {
		m.updateCopyHighlight(false)
	} else if m.searchQuery != "" {
		m.executeSearch(m.searchQuery)
	} else if m.linkFocusActive {
		m.updateLinkHighlight(false)
	} else {
		m.viewport.SetContent(entry.rendered)
	}

	return nil
}
