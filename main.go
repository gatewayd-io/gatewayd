package main

import (
	"github.com/gatewayd-io/gatewayd/network"
	"github.com/panjf2000/gnet/v2"
)

func main() {
	// Create a PostgreSQL server.
	server := &network.PostgreSQLServer{
		Address: ":15432",
		Options: []gnet.Option{
			gnet.WithMulticore(true),
			gnet.WithReusePort(true),
			gnet.WithTicker(false),
		},
	}
	server.Run()
}
