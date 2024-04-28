package cmd

import (
	"github.com/gatewayd-io/gatewayd/plugin"
	"github.com/spf13/cobra"
)

var (
	pluginScaffoldInputFile string
	pluginScaffoldOutputDir string
)

// pluginScaffoldCmd represents the scaffold command.
var pluginScaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "Scaffold a plugin and store the files into a directory",
	Run: func(cmd *cobra.Command, _ []string) {
		createdFiles, err := plugin.Scaffold(pluginScaffoldInputFile, pluginScaffoldOutputDir)
		if err != nil {
			cmd.Println("Scaffold failed: ", err)
			return
		}

		cmd.Println("scaffold done")
		cmd.Println("created files:")
		for _, file := range createdFiles {
			cmd.Println(file)
		}
	},
}

func init() {
	pluginCmd.AddCommand(pluginScaffoldCmd)
	pluginScaffoldCmd.Flags().StringVarP(
		&pluginScaffoldInputFile,
		"input-file", "i", "input.yaml",
		"Plugin scaffold input file")
	pluginScaffoldCmd.Flags().StringVarP(
		&pluginScaffoldOutputDir,
		"output-dir", "o", "./scaffolds",
		"Output directory for the scaffold")
}
