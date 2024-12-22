package api

import (
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/gatewayd-io/gatewayd/raft"
)

func liveness(servers map[string]*network.Server, raftNode *raft.Node) bool {
	for _, v := range servers {
		if !v.IsRunning() {
			return false
		}
	}
	if raftNode != nil {
		if !raftNode.GetHealthStatus().IsHealthy {
			return false
		}
	}
	return true
}
