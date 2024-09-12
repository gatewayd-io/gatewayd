package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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

// mustPullPlugin pulls the gatewayd-plugin-cache plugin and returns the path to the archive.
func mustPullPlugin() (string, error) {
	pluginURL := "github.com/gatewayd-io/gatewayd-plugin-cache@v0.2.10"

	fileName := fmt.Sprintf(
		"./gatewayd-plugin-cache-%s-%s-%s%s",
		runtime.GOOS,
		runtime.GOARCH,
		"v0.2.10",
		getFileExtension(),
	)

	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		_, err := executeCommandC(rootCmd, "plugin", "install", "--pull-only", pluginURL)
		if err != nil {
			return "", err
		}
	}

	return filepath.Abs(fileName) //nolint:wrapcheck
}

// setupTestContainer initializes and starts the PostgreSQL test container.
func setupPostgreSQLTestContainer(ctx context.Context, t *testing.T) (string, nat.Port) {
	t.Helper()

	postgresPort := "5432"
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:latest",
			ExposedPorts: []string{postgresPort + "/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
			},
			WaitingFor: wait.ForAll(
				wait.ForLog("database system is ready to accept connections"),
				wait.ForListeningPort(nat.Port(postgresPort+"/tcp")),
			),
		},
		Started: true,
	})
	require.NoError(t, err)

	hostIP, err := container.Host(ctx)
	require.NoError(t, err, "Failed to retrieve PostgreSQL test container host IP")

	mappedPort, err := container.MappedPort(ctx, nat.Port(postgresPort+"/tcp"))
	require.NoError(t, err, "Failed to map PostgreSQL test container port")

	return hostIP, mappedPort
}
