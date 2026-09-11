package ui

import (
	"regexp"
	"strings"
)

// Markdownの見出し位置（行番号）を解析・保持する
func (m *model) updateHeadingLines() {
	re := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	matches := re.FindAllStringSubmatch(m.content, -1)

	var hLines []int
	searchStart := 0

	for _, match := range matches {
		title := strings.TrimSpace(match[2])
		for i := searchStart; i < len(m.renderedLines); i++ {
			if strings.Contains(m.renderedLines[i], title) {
				hLines = append(hLines, i)
				searchStart = i + 1
				break
			}
		}
	}
	m.headingLines = hLines
}
