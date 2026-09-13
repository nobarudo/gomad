package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nobarudo/gomad/internal/ui"
)

// ↓ フラグ変数定義
var (
	styleFlag string
	watchFlag bool
)

var rootCmd = &cobra.Command{
	Use:   "gomad <file.md>",
	Short: "A simple terminal Markdown viewer",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		return ui.Run(filePath, styleFlag, watchFlag)
	},
}

func init() {
	// -s / --style フラグを追加 (デフォルト値: tokyo-night)
	rootCmd.Flags().StringVarP(&styleFlag, "style", "s", "dark", "Markdown color style (tokyo-night, dracula, dark, light, pink, notty)")
	// -w / --watch フラグを追加 (デフォルト値: true)
	rootCmd.Flags().BoolVarP(&watchFlag, "watch", "w", true, "Enable auto-reload on file change")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
