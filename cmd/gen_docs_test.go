package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_genDocs(t *testing.T) {
	_, err := executeCommandC(rootCmd, "gen-docs", "--output-dir", "./docs")
	require.NoError(t, err, "genDocs should not return an error")
	assert.DirExists(t, "./docs", "genDocs should create the output directory")
	assert.FileExists(t, "./docs/gatewayd.md", "genDocs should create the markdown file")
	require.NoError(t, os.RemoveAll("./docs"))
}
