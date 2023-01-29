package main

import "github.com/mostafa/gatewayd/network"

func main() {
	err := network.NewListener(&network.ListenerCfg{
		Protocol:    "tcp",
		Address:     ":15432", // incoming port
		ConnHandler: network.ProxyHandler,
		DialerCfg: &network.DialerCfg{
			ZeroCopy: true,
			Protocol: "tcp",
			Address:  ":5432", // database port
		},
	})
	if err != nil {
		panic(err)
	}
}
