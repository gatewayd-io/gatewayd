package act

import (
	"time"

	sdkAct "github.com/gatewayd-io/gatewayd-plugin-sdk/act"
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
