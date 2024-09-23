package testhelpers

import (
	"context"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupPostgreSQLTestContainer initializes and starts the PostgreSQL test container.
func SetupPostgreSQLTestContainer(ctx context.Context, t *testing.T) (string, nat.Port) {
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
