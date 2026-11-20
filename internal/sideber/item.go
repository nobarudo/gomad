package sideber

import (
	"regexp"
	"strings"
)

// Item は目次（見出し）の1項目を表します
type Item struct {
	Level int    // 見出しレベル (1〜6: # の数)
	Title string // 見出しタイトル
	Line  int    // レンダリング後の行番号 (本文中の行位置)
}

var (
	headingRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
)

func stripANSI(str string) string {
	idx := strings.IndexByte(str, '\x1b')
	if idx == -1 {
		return str
	}

	var b strings.Builder
	b.Grow(len(str))
	b.WriteString(str[:idx])

	n := len(str)
	for i := idx; i < n; {
		if str[i] == '\x1b' && i+1 < n && str[i+1] == '[' {
			j := i + 2
			for j < n && ((str[j] >= '0' && str[j] <= '9') || str[j] == ';') {
				j++
			}
			if j < n && ((str[j] >= 'a' && str[j] <= 'z') || (str[j] >= 'A' && str[j] <= 'Z')) {
				i = j + 1
				continue
			}
		}
		b.WriteByte(str[i])
		i++
	}
	return b.String()
}

// ParseItems はMarkdownテキストから見出しを抽出し、レンダリング後の行番号と紐付けます
func ParseItems(content string, renderedLines []string) []Item {
	// Glamourが付与したANSIエスケープシーケンス（カラーコード等）を除去したプレーン行を用意
	plainLines := make([]string, len(renderedLines))
	for i, rl := range renderedLines {
		plainLines[i] = stripANSI(rl)
	}
	return ParseItemsWithPlainLines(content, plainLines)
}

// ParseItemsWithPlainLines は既にANSI除去済みの行配列を使って見出しを抽出します
func ParseItemsWithPlainLines(content string, plainLines []string) []Item {
	lines := strings.Split(content, "\n")
	var items []Item
	inCodeBlock := false

	searchStart := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// コードブロック内の # を見出しとして誤検知しないようにスキップ
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		matches := headingRegex.FindStringSubmatch(trimmed)
		if len(matches) == 3 {
			level := len(matches[1])
			title := strings.TrimSpace(matches[2])
			targetLine := -1

			// プレーン行から見出し位置を探す
			for i := searchStart; i < len(plainLines); i++ {
				if strings.Contains(plainLines[i], title) {
					targetLine = i
					searchStart = i + 1
					break
				}
			}

			// もし完全一致で見つからなかった場合、インライン記号（バッククォート等）を除去して再検索
			if targetLine == -1 {
				cleanTitle := strings.Map(func(r rune) rune {
					if strings.ContainsRune("`*_{}[]()#+-.", r) {
						return -1
					}
					return r
				}, title)
				cleanTitle = strings.TrimSpace(cleanTitle)
				if cleanTitle != "" {
					for i := searchStart; i < len(plainLines); i++ {
						if strings.Contains(plainLines[i], cleanTitle) {
							targetLine = i
							searchStart = i + 1
							break
						}
					}
				}
			}

			// それでも見つからなかった場合は直前の見つかった行（または 0）を設定してインデックス欠落を防ぐ
			if targetLine == -1 {
				if len(items) > 0 && items[len(items)-1].Line >= 0 {
					targetLine = items[len(items)-1].Line
				} else {
					targetLine = 0
				}
			}

			items = append(items, Item{
				Level: level,
				Title: title,
				Line:  targetLine,
			})
		}
	}

	return items
}
