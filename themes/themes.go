package themes

import (
	"embed"
	"strings"
)

// ThemeFS は themes ディレクトリ内のすべての .json ファイルをバイナリに埋め込みます
//
//go:embed *.json
var ThemeFS embed.FS

// BuiltinStyles はGlamourに標準で組み込まれているスタイル一覧です
var BuiltinStyles = []string{
	"tokyo-night",
	"dracula",
	"dark",
	"light",
	"pink",
	"notty",
}

// AvailableStyles は組み込みスタイルと themes/*.json から自動検出した全スタイル名の一覧を返します
func AvailableStyles() []string {
	styles := make([]string, len(BuiltinStyles))
	copy(styles, BuiltinStyles)

	entries, err := ThemeFS.ReadDir(".")
	if err != nil {
		return styles
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		exists := false
		for _, s := range styles {
			if s == name {
				exists = true
				break
			}
		}
		if !exists {
			styles = append(styles, name)
		}
	}

	return styles
}

// GetThemeJSON は指定されたテーマ名のJSONバイト列を返します。
// themes/*.json に存在しない（Glamour標準スタイルの）場合は nil, false を返します。
func GetThemeJSON(name string) ([]byte, bool) {
	filename := name + ".json"
	data, err := ThemeFS.ReadFile(filename)
	if err != nil {
		return nil, false
	}
	return data, true
}
