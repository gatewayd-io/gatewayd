package network

import (
	"errors"
	"os"
	"sync"

	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
)

type Traffic func(buf []byte, err error) error

type Proxy interface {
	Connect(c gnet.Conn) error
	Disconnect(c gnet.Conn) error
	PassThrough(c gnet.Conn, onIncomingTraffic, onOutgoingTraffic Traffic) error
	Reconnect(cl *Client) *Client
	Shutdown()
	Size() int
}

type ProxyImpl struct {
	pool        Pool
	connClients sync.Map

	PoolSize            int
	Elastic             bool
	ReuseElasticClients bool
}

var _ Proxy = &ProxyImpl{}

func NewProxy(size int, elastic, reuseElasticClients bool) *ProxyImpl {
	proxy := ProxyImpl{
		pool:        NewPool(),
		connClients: sync.Map{},

		PoolSize:            size,
		Elastic:             elastic,
		ReuseElasticClients: reuseElasticClients,
	}

	if !proxy.Elastic {
		for i := 0; i < size; i++ {
			client := NewClient("tcp", "localhost:5432", 4096)
			if client != nil {
				if err := proxy.pool.Put(client); err != nil {
					logrus.Panic(err)
				}
			}
		}

		logrus.Infof("There are %d clients in the pool", len(proxy.pool.ClientIDs()))
		if len(proxy.pool.ClientIDs()) != size {
			logrus.Error("The pool size is incorrect, either because the clients are cannot connect (no network connectivity) or the server is not running")
			os.Exit(1)
		}
	}

	return &proxy
}

func (pr *ProxyImpl) Connect(c gnet.Conn) error {
	clientIDs := pr.pool.ClientIDs()

	var client *Client
	if len(clientIDs) == 0 {
		// Pool is exhausted
		if pr.Elastic {
			// Create a new client
			client = NewClient("tcp", "localhost:5432", 4096)
			logrus.Debugf("Reused the client %s by putting it back in the pool", client.ID)
		} else {
			return errors.New("pool is exhausted")
		}
	} else {
		// Get a client from the pool
		logrus.Debugf("Available clients: %v", len(clientIDs))
		client = pr.pool.Pop(clientIDs[0])
	}

	if client.ID != "" {
		pr.connClients.Store(c, client)
		logrus.Debugf("Client %s has been assigned to %s", client.ID, c.RemoteAddr().String())
	} else {
		return errors.New("client is not connected (connect)")
	}

	logrus.Debugf("[C] There are %d clients in the pool", len(pr.pool.ClientIDs()))
	logrus.Debugf("[C] There are %d clients in use", pr.Size())

	return nil
}

func (pr *ProxyImpl) Disconnect(c gnet.Conn) error {
	var client *Client
	if cl, ok := pr.connClients.Load(c); ok {
		client = cl.(*Client)
	}
	pr.connClients.Delete(c)

	// TODO: The connection is unstable when I put the client back in the pool
	// If the client is not in the pool, put it back
	if pr.Elastic && pr.ReuseElasticClients || !pr.Elastic {
		client = pr.Reconnect(client)
		if client != nil && client.ID != "" {
			if err := pr.pool.Put(client); err != nil {
				return err
			}
		}
	} else {
		client.Close()
	}

	logrus.Debugf("[D] There are %d clients in the pool", len(pr.pool.ClientIDs()))
	logrus.Debugf("[D] There are %d clients in use", pr.Size())

	return nil
}

func (pr *ProxyImpl) PassThrough(c gnet.Conn, onIncomingTraffic, onOutgoingTraffic Traffic) error {
	// TODO: Handle bi-directional traffic
	// Currently the passthrough is a one-way street from the client to the server, that is,
	// the client can send data to the server and receive the response back, but the server
	// cannot take initiative and send data to the client. So, there should be another event-loop
	// that listens for data from the server and sends it to the client

	var client *Client
	if c, ok := pr.connClients.Load(c); ok {
		client = c.(*Client)
	} else {
		return errors.New("client not found")
	}

	// buf contains the data from the client (query)
	buf, err := c.Next(-1)
	if err != nil {
		logrus.Errorf("Error reading from client: %v", err)
	}
	if err = onIncomingTraffic(buf, err); err != nil {
		logrus.Errorf("Error processing data from client: %v", err)
	}

	// TODO: parse the buffer and send the response or error
	// TODO: This is a very basic implementation of the gateway
	// and it is synchronous. I should make it asynchronous.
	logrus.Debugf("Received %d bytes from %s", len(buf), c.RemoteAddr().String())

	// Send the query to the server
	err = client.Send(buf)
	if err != nil {
		return err
	}

	// Receive the response from the server
	size, response, err := client.Receive()
	if err := onOutgoingTraffic(response[:size], err); err != nil {
		logrus.Errorf("Error processing data from server: %v", err)
	}

	if err != nil {
		// FIXME: Is this the right way to handle this error?
		if err.Error() == "EOF" {
			logrus.Error("The client is not connected to the server anymore")
			// Either the client is not connected to the server anymore or
			// server forceful closed the connection
			// Reconnect the client
			client = pr.Reconnect(client)
			// Store the client in the map, replacing the old one
			pr.connClients.Store(c, client)
		} else {
			// Write the error to the client
			_, err := c.Write(response[:size])
			if err != nil {
				logrus.Errorf("Error writing the error to client: %v", err)
			}
		}
	} else {
		// Write the response to the incoming connection
		_, err := c.Write(response[:size])
		if err != nil {
			logrus.Errorf("Error writing to client: %v", err)
		}
	}

	return nil
}

func (pr *ProxyImpl) Reconnect(cl *Client) *Client {
	// Close the client
	if cl != nil && cl.ID != "" {
		cl.Close()
	}
	return NewClient("tcp", "localhost:5432", 4096)
}

func (pr *ProxyImpl) Shutdown() {
	pr.pool.Shutdown()
	logrus.Debug("All busy client connections have been closed")

	availableClients := pr.pool.ClientIDs()
	for _, clientID := range availableClients {
		client := pr.pool.Pop(clientID)
		client.Close()
	}
	logrus.Debug("All available client connections have been closed")
}

func (pr *ProxyImpl) Size() int {
	var size int
	pr.connClients.Range(func(_, _ interface{}) bool {
		size++
		return true
	})

	return size
}
