package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gomad/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "mdview <file.md>",
	Short: "A simple terminal Markdown viewer",
	Args:  cobra.ExactArgs(1), // 引数（ファイルパス）を1つだけ必須にする
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		return ui.Run(filePath)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
