package network

import (
	"errors"

	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
)

type Proxy interface {
	Connect(c gnet.Conn) error
	Disconnect(c gnet.Conn) error
	PassThrough(c gnet.Conn) error
	Shutdown()
}

type ProxyImpl struct {
	pool             Pool
	connectedClients map[gnet.Conn]Client
	PoolSize         int
}

var _ Proxy = &ProxyImpl{}

func NewProxy(size int) *ProxyImpl {
	proxy := ProxyImpl{
		pool:             NewPool(),
		connectedClients: make(map[gnet.Conn]Client),
		PoolSize:         size,
	}

	for i := 0; i < size; i++ {
		c := NewClient("tcp", "localhost:5432", 4096)
		if err := proxy.pool.Put(c); err != nil {
			logrus.Panic(err)
		}
	}

	return &proxy
}

func (pr *ProxyImpl) Connect(c gnet.Conn) error {
	clientIDs := pr.pool.GetClientIDs()
	if len(clientIDs) == 0 {
		logrus.Error("No clients available")
		return errors.New("no clients available")
	}

	client := pr.pool.Pop(clientIDs[0])
	logrus.Infof("Client %s has been assigned to %s", client.ID, c.RemoteAddr().String())
	pr.connectedClients[c] = client

	return nil
}

func (pr *ProxyImpl) Disconnect(c gnet.Conn) error {
	client := pr.connectedClients[c]
	pr.pool.Put(client)
	delete(pr.connectedClients, c)

	return nil
}

func (pr *ProxyImpl) PassThrough(c gnet.Conn) error {
	// buf contains the data from the client (query)
	buf, _ := c.Next(-1)

	// TODO: parse the buffer and send the response or error
	// TODO: This is a very basic implementation of the gateway
	// and it is synchronous. I should make it asynchronous.
	logrus.Infof("Received %d bytes from %s", len(buf), c.RemoteAddr().String())

	// Send the query to the server
	pr.connectedClients[c].Send(buf)
	// Receive the response from the server
	size, response := pr.connectedClients[c].Receive()
	// Write the response to the incoming connection
	c.Write(response[:size])

	return nil
}

func (pr *ProxyImpl) Shutdown() {
	for _, client := range pr.connectedClients {
		client.Close()
	}
	logrus.Info("All busy client connections have been closed")

	availableClients := pr.pool.GetClientIDs()
	for _, clientID := range availableClients {
		client := pr.pool.Pop(clientID)
		client.Close()
	}
	logrus.Info("All available client connections have been closed")
}
