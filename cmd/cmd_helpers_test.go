package cmd

import (
	"bytes"

	"github.com/spf13/cobra"
)

// executeCommandC executes a cobra command and returns the command, output, and error.
// Taken from https://github.com/spf13/cobra/blob/0c72800b8dba637092b57a955ecee75949e79a73/command_test.go#L48.
func executeCommandC(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	_, err := root.ExecuteC()

	return buf.String(), err
}
