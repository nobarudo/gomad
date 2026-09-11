package ui

type segment struct {
	text   string
	isANSI bool
}

// ANSIエスケープシーケンスと通常の文字に分解する（ルーン対応）
func parseSegments(ansiStr string) []segment {
	var segments []segment
	runes := []rune(ansiStr)
	n := len(runes)
	i := 0
	for i < n {
		if runes[i] == '\x1b' {
			start := i
			i++ // skip '\x1b'
			for i < n {
				r := runes[i]
				i++
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
					break
				}
			}
			segments = append(segments, segment{text: string(runes[start:i]), isANSI: true})
		} else {
			segments = append(segments, segment{text: string(runes[i]), isANSI: false})
			i++
		}
	}
	return segments
}
