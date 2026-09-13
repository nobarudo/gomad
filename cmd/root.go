package cmd

import (
	"fmt"
	"io"
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
	Use:   "gomad [file.md]",
	Short: "A simple terminal Markdown viewer",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && args[0] != "-" {
			filePath := args[0]
			return ui.RunFile(filePath, styleFlag, watchFlag)
		}

		// 標準入力がパイプであるか確認
		stat, err := os.Stdin.Stat()
		if err != nil {
			return err
		}
		// パイプでなく端末からの直接入力の場合（引数なしで実行された場合）
		if (stat.Mode()&os.ModeCharDevice) != 0 && len(args) == 0 {
			return cmd.Help()
		}

		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from standard input: %w", err)
		}

		return ui.RunStdin(string(content), styleFlag)
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
