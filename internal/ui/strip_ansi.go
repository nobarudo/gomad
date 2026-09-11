package ui

import "regexp"

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// ANSIエスケープコードを除去するヘルパー
func stripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}
