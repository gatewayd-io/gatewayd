package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var docOutputDir string

var genDocs = &cobra.Command{
	Use:    "gen-docs",
	Short:  "Generate markdown documentation",
	Hidden: true,
	Run: func(cmd *cobra.Command, _ []string) {
		err := doc.GenMarkdownTree(rootCmd, docOutputDir)
		if err != nil {
			cmd.PrintErr(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(genDocs)

	genDocs.Flags().StringVarP(
		&docOutputDir,
		"output-dir", "o", "./docs",
		"Output directory for markdown files")
}
