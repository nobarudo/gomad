package ui

import "strings"

// stripANSI はANSIエスケープシーケンス（CSIシーケンス）を高速に除去します。
// エスケープシーケンスを含まない場合は割り当てなしで元の文字列を返します。
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
