package api

import (
	"github.com/gatewayd-io/gatewayd/network"
)

func liveness(servers map[string]*network.Server) bool {
	for _, v := range servers {
		if !v.IsRunning() {
			return false
		}
	}
	return true
}
