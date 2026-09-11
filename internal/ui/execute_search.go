package ui

import "strings"

// 検索ワードで各行を検索する
func (m *model) executeSearch(query string) {
	m.searchQuery = query
	m.searchResults = nil
	m.currentMatchIndex = 0

	if query == "" {
		if len(m.pristineLines) > 0 {
			m.renderedLines = make([]string, len(m.pristineLines))
			copy(m.renderedLines, m.pristineLines)
			m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
		}
		return
	}

	lowerQuery := strings.ToLower(query)
	for i, line := range m.pristineLines {
		plain := strings.ToLower(stripANSI(line))
		if strings.Contains(plain, lowerQuery) {
			m.searchResults = append(m.searchResults, i)
		}
	}

	m.updateHighlight()
}

func highlightQueryInLine(ansiStr string, query string, highlightStart, highlightEnd string) string {
	if query == "" {
		return ansiStr
	}

	segments := parseSegments(ansiStr)

	var plainBuilder strings.Builder
	var plainToSeg []int

	for segIdx, seg := range segments {
		if !seg.isANSI {
			plainBuilder.WriteString(seg.text)
			plainToSeg = append(plainToSeg, segIdx)
		}
	}

	plainText := plainBuilder.String()
	plainRunes := []rune(plainText)
	lowerPlainRunes := []rune(strings.ToLower(plainText))
	lowerQueryRunes := []rune(strings.ToLower(query))

	queryLen := len(lowerQueryRunes)
	if queryLen == 0 || len(plainRunes) < queryLen {
		return ansiStr
	}

	// ルーンスライスでの検索
	var matchStarts []int
	n := len(lowerPlainRunes)
	m := len(lowerQueryRunes)
	for i := 0; i <= n-m; i++ {
		match := true
		for j := 0; j < m; j++ {
			if lowerPlainRunes[i+j] != lowerQueryRunes[j] {
				match = false
				break
			}
		}
		if match {
			matchStarts = append(matchStarts, i)
			i += m - 1 // 重複防止
		}
	}

	if len(matchStarts) == 0 {
		return ansiStr
	}

	for _, start := range matchStarts {
		segStart := plainToSeg[start]
		segEnd := plainToSeg[start+queryLen-1]

		segments[segStart].text = highlightStart + segments[segStart].text
		segments[segEnd].text = segments[segEnd].text + highlightEnd
	}

	var result strings.Builder
	for _, seg := range segments {
		result.WriteString(seg.text)
	}
	return result.String()
}
