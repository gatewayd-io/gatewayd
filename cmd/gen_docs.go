package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var docOutputDir string

var genDocs = &cobra.Command{
	Use:    "gen-docs",
	Short:  "Generate markdown documentation",
	Hidden: true,
	Run: func(cmd *cobra.Command, _ []string) {
		// Create the output directory if it doesn't exist
		if err := os.MkdirAll(docOutputDir, 0o755); err != nil {
			cmd.PrintErr(err)
			return
		}
		// Generate the markdown files
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
