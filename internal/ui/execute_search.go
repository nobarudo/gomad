package ui

import (
	"strings"
	"unicode/utf8"
)

type runeSpan struct {
	start int
	end   int
}

// 検索ワードで各行を検索する
func (m *model) executeSearch(query string) {
	m.searchQuery = query
	m.searchResults = nil
	m.currentMatchIndex = 0
	if query != "" {
		m.linkFocusActive = false
	}

	if query == "" {
		if len(m.pristineLines) > 0 {
			m.renderedLines = make([]string, len(m.pristineLines))
			copy(m.renderedLines, m.pristineLines)
			m.viewport.SetContent(strings.Join(m.renderedLines, "\n"))
		}
		return
	}

	lowerQuery := strings.ToLower(query)
	usePlain := len(m.plainLines) == len(m.pristineLines)
	for i, line := range m.pristineLines {
		var plain string
		if usePlain {
			plain = m.plainLines[i]
		} else {
			plain = stripANSI(line)
		}
		if strings.Contains(strings.ToLower(plain), lowerQuery) {
			m.searchResults = append(m.searchResults, i)
		}
	}

	m.updateHighlight()
}

// applyHighlightSpansToANSI は ANSI エスケープシーケンスを保持したまま、
// 可視ルーン位置の範囲リストに対してハイライトコードを単一パスで挿入します。
func applyHighlightSpansToANSI(ansiStr string, spans []runeSpan, highlightStart, highlightEnd string) string {
	if len(spans) == 0 {
		return ansiStr
	}

	var b strings.Builder
	b.Grow(len(ansiStr) + len(spans)*(len(highlightStart)+len(highlightEnd)))

	spanIdx := 0
	visibleRune := 0
	n := len(ansiStr)

	for i := 0; i < n; {
		if ansiStr[i] == '\x1b' {
			start := i
			i++
			for i < n {
				c := ansiStr[i]
				i++
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
					break
				}
			}
			b.WriteString(ansiStr[start:i])
			continue
		}

		_, size := utf8.DecodeRuneInString(ansiStr[i:])

		if spanIdx < len(spans) && visibleRune == spans[spanIdx].start {
			b.WriteString(highlightStart)
		}

		b.WriteString(ansiStr[i : i+size])

		if spanIdx < len(spans) && visibleRune == spans[spanIdx].end-1 {
			b.WriteString(highlightEnd)
			spanIdx++
		}

		visibleRune++
		i += size
	}

	return b.String()
}

func highlightQueryInLine(ansiStr string, query string, highlightStart, highlightEnd string) string {
	if query == "" {
		return ansiStr
	}

	plainText := stripANSI(ansiStr)
	lowerPlain := strings.ToLower(plainText)
	lowerQuery := strings.ToLower(query)

	firstIdx := strings.Index(lowerPlain, lowerQuery)
	if firstIdx == -1 {
		return ansiStr
	}

	queryRuneLen := utf8.RuneCountInString(lowerQuery)
	if queryRuneLen == 0 {
		return ansiStr
	}

	var spans []runeSpan
	if utf8.RuneCountInString(lowerPlain) == utf8.RuneCountInString(plainText) {
		// 高速パス: strings.Index で非重複マッチを収集
		bytePos := 0
		runePos := 0
		for {
			idx := strings.Index(lowerPlain[bytePos:], lowerQuery)
			if idx == -1 {
				break
			}
			matchStartRune := runePos + utf8.RuneCountInString(lowerPlain[bytePos:bytePos+idx])
			spans = append(spans, runeSpan{
				start: matchStartRune,
				end:   matchStartRune + queryRuneLen,
			})
			bytePos += idx + len(lowerQuery)
			runePos = matchStartRune + queryRuneLen
		}
	} else {
		// 稀なUnicode大文字小文字変換でルーン数が変わる場合のフォールバック
		lowerPlainRunes := []rune(lowerPlain)
		lowerQueryRunes := []rune(lowerQuery)
		pn := len(lowerPlainRunes)
		qn := len(lowerQueryRunes)
		for i := 0; i <= pn-qn; i++ {
			match := true
			for j := 0; j < qn; j++ {
				if lowerPlainRunes[i+j] != lowerQueryRunes[j] {
					match = false
					break
				}
			}
			if match {
				spans = append(spans, runeSpan{start: i, end: i + qn})
				i += qn - 1
			}
		}
	}

	return applyHighlightSpansToANSI(ansiStr, spans, highlightStart, highlightEnd)
}
