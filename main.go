package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/panjf2000/gnet/v2"
)

// import "github.com/gatewayd-io/gatewayd/network"

// func main() {
// 	err := network.NewListener(&network.ListenerCfg{
// 		Protocol:    "tcp",
// 		Address:     ":15432", // incoming port
// 		ConnHandler: network.ProxyHandler,
// 		DialerCfg: &network.DialerCfg{
// 			ZeroCopy: true,
// 			Protocol: "tcp",
// 			Address:  ":5432", // database port
// 		},
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// }

type echoServer struct {
	gnet.BuiltinEventEngine

	eng       gnet.Engine
	addr      string
	multicore bool
}

func (es *echoServer) OnBoot(eng gnet.Engine) gnet.Action {
	es.eng = eng
	log.Printf("echo server with multi-core=%t is listening on %s\n", es.multicore, es.addr)
	return gnet.None
}

func (es *echoServer) OnTraffic(c gnet.Conn) gnet.Action {
	buf, _ := c.Next(-1)
	c.Write(buf)
	log.Printf("total connections: %d", es.eng.CountConnections())
	return gnet.None
}

func main() {
	var port int
	var multicore bool

	// Example command: go run echo.go --port 9000 --multicore=true
	flag.IntVar(&port, "port", 9000, "--port 9000")
	flag.BoolVar(&multicore, "multicore", false, "--multicore true")
	flag.Parse()
	echo := &echoServer{
		addr:      fmt.Sprintf("tcp://:%d", port),
		multicore: multicore,
	}
	log.Fatal(
		gnet.Run(
			echo,
			echo.addr,
			gnet.WithMulticore(multicore),
		),
	)
}
