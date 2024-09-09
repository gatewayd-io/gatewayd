package act

import (
	"context"
	"testing"
	"time"

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
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:6",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	redisContainer, err := testcontainers.GenericContainer(
		ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, redisContainer.Terminate(ctx))
	})
	host, err := redisContainer.Host(ctx)
	assert.NoError(t, err)
	port, err := redisContainer.MappedPort(ctx, "6379/tcp")
	assert.NoError(t, err)
	return host + ":" + port.Port()
}
