package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gomad/internal/ui" // ※ ご自身のモジュール名に合わせて変更してください
)

// ↓ この変数定義が必要です！
var styleFlag string

var rootCmd = &cobra.Command{
	Use:   "mdview <file.md>",
	Short: "A simple terminal Markdown viewer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		return ui.Run(filePath, styleFlag)
	},
}

func init() {
	// -s / --style フラグを追加 (デフォルト値: tokyonight)
	rootCmd.Flags().StringVarP(&styleFlag, "style", "s", "tokyo-night", "Markdown color style (tokyo-night, dracula, dark, light, pink, notty)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
