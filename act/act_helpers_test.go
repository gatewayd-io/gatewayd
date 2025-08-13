package act

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func createWaitActEntities(async bool) (
	string,
	map[string]*sdkAct.Action,
	map[string]*sdkAct.Signal,
	map[string]*sdkAct.Policy,
) {
	name := "waitSync"
	if async {
		name = "waitAsync"
	}
	actions := map[string]*sdkAct.Action{
		name: {
			Name:     name,
			Metadata: nil,
			Sync:     !async,
			Terminal: false,
			Run: func(_ map[string]any, _ ...sdkAct.Parameter) (any, error) {
				time.Sleep(1 * time.Second)
				return true, nil
			},
		},
	}
	signals := map[string]*sdkAct.Signal{
		name: {
			Name: name,
			Metadata: map[string]any{
				"log":     true,
				"level":   "info",
				"message": "test",
				"async":   async,
			},
		},
	}
	policy := map[string]*sdkAct.Policy{
		name: sdkAct.MustNewPolicy(
			name,
			`true`,
			map[string]any{"log": "enabled"},
		),
	}

	return name, actions, signals, policy
}

func createTestRedis(t *testing.T) string {
	t.Helper()
	ctx := t.Context()

	redisPort := "6379"

	redisContainer, err := testcontainers.GenericContainer(
		ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "redis:7",
				ExposedPorts: []string{redisPort + "/tcp"},
				WaitingFor: wait.ForAll(
					wait.ForLog("Ready to accept connections"),
					wait.ForListeningPort(nat.Port(redisPort+"/tcp")),
				),
			},
			Started: true,
			Logger:  log.New(io.Discard, "", 0),
		},
	)
	assert.NoError(t, err)

	host, err := redisContainer.Host(ctx)
	assert.NoError(t, err)

	port, err := redisContainer.MappedPort(ctx, nat.Port(redisPort+"/tcp"))
	assert.NoError(t, err)

	return host + ":" + port.Port()
}
