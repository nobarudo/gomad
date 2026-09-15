package themes

import (
	"strings"
	"testing"
)

func TestAvailableStyles(t *testing.T) {
	styles := AvailableStyles()

	// 組み込みスタイルが含まれていること
	for _, expected := range BuiltinStyles {
		found := false
		for _, s := range styles {
			if s == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected AvailableStyles to include builtin style %q", expected)
		}
	}

	// 自動検出された cyan が含まれていること
	foundCyan := false
	for _, s := range styles {
		if s == "cyan" {
			foundCyan = true
			break
		}
	}
	if !foundCyan {
		t.Errorf("Expected AvailableStyles to include 'cyan'")
	}
}

func TestGetThemeJSON(t *testing.T) {
	// cyan.json の読み込み
	data, ok := GetThemeJSON("cyan")
	if !ok || len(data) == 0 {
		t.Fatalf("Expected GetThemeJSON('cyan') to succeed")
	}
	if !strings.Contains(string(data), "#7dcfff") {
		t.Errorf("Expected cyan.json to contain color '#7dcfff'")
	}

	// 組み込みスタイルは themes/*.json にはないので false
	_, ok = GetThemeJSON("dark")
	if ok {
		t.Errorf("Expected GetThemeJSON('dark') to return false for builtin styles")
	}

	// 存在しないスタイル
	_, ok = GetThemeJSON("non_existent_theme")
	if ok {
		t.Errorf("Expected GetThemeJSON('non_existent_theme') to return false")
	}
}
