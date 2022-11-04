package network

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
)

type Traffic func(gconn gnet.Conn, cl *Client, buf []byte, err error) error

type Proxy interface {
	Connect(gconn gnet.Conn) error
	Disconnect(gconn gnet.Conn) error
	PassThrough(gconn gnet.Conn, onIncomingTraffic, onOutgoingTraffic Traffic) error
	Reconnect(cl *Client) *Client
	Shutdown()
	Size() int
}

type ProxyImpl struct {
	pool    Pool
	clients sync.Map

	Elastic             bool
	ReuseElasticClients bool

	// ClientConfig is used for elastic proxy and reconnection
	ClientConfig *Client
}

var _ Proxy = &ProxyImpl{}

func NewProxy(
	pool Pool, elastic, reuseElasticClients bool, clientConfig *Client) *ProxyImpl {
	proxy := ProxyImpl{
		clients:             sync.Map{},
		Elastic:             elastic,
		ReuseElasticClients: reuseElasticClients,
		ClientConfig:        clientConfig,
	}

	if pool != nil {
		proxy.pool = pool
	} else {
		proxy.pool = NewPool()
	}

	return &proxy
}

func (pr *ProxyImpl) Connect(gconn gnet.Conn) error {
	clientIDs := pr.pool.ClientIDs()

	var client *Client
	if len(clientIDs) == 0 {
		// Pool is exhausted
		if pr.Elastic {
			// Create a new client
			client = NewClient(
				pr.ClientConfig.Network,
				pr.ClientConfig.Address,
				pr.ClientConfig.ReceiveBufferSize,
			)
			logrus.Debugf("Reused the client %s by putting it back in the pool", client.ID)
		} else {
			return ErrPoolExhausted
		}
	} else {
		// Get a client from the pool
		logrus.Debugf("Available clients: %v", len(clientIDs))
		client = pr.pool.Pop(clientIDs[0])
	}

	if client.ID != "" {
		pr.clients.Store(gconn, client)
		logrus.Debugf("Client %s has been assigned to %s", client.ID, gconn.RemoteAddr().String())
	} else {
		return ErrClientNotConnected
	}

	logrus.Debugf("[C] There are %d clients in the pool", len(pr.pool.ClientIDs()))
	logrus.Debugf("[C] There are %d clients in use", pr.Size())

	return nil
}

func (pr *ProxyImpl) Disconnect(gconn gnet.Conn) error {
	var client *Client
	if cl, ok := pr.clients.Load(gconn); ok {
		if c, ok := cl.(*Client); ok {
			client = c
		}
	}
	pr.clients.Delete(gconn)

	// TODO: The connection is unstable when I put the client back in the pool
	// If the client is not in the pool, put it back
	if pr.Elastic && pr.ReuseElasticClients || !pr.Elastic {
		client = pr.Reconnect(client)
		if client != nil && client.ID != "" {
			if err := pr.pool.Put(client); err != nil {
				return fmt.Errorf("failed to put the client back in the pool: %w", err)
			}
		}
	} else {
		client.Close()
	}

	logrus.Debugf("[D] There are %d clients in the pool", len(pr.pool.ClientIDs()))
	logrus.Debugf("[D] There are %d clients in use", pr.Size())

	return nil
}

//nolint:funlen
func (pr *ProxyImpl) PassThrough(gconn gnet.Conn, onIncomingTraffic, onOutgoingTraffic Traffic) error {
	// TODO: Handle bi-directional traffic
	// Currently the passthrough is a one-way street from the client to the server, that is,
	// the client can send data to the server and receive the response back, but the server
	// cannot take initiative and send data to the client. So, there should be another event-loop
	// that listens for data from the server and sends it to the client

	var client *Client
	if c, ok := pr.clients.Load(gconn); ok {
		if cl, ok := c.(*Client); ok {
			client = cl
		}
	} else {
		return ErrClientNotFound
	}

	// buf contains the data from the client (query)
	buf, err := gconn.Next(-1)
	if err != nil {
		logrus.Errorf("Error reading from client: %v", err)
	}
	if err = onIncomingTraffic(gconn, client, buf, err); err != nil {
		logrus.Errorf("Error processing data from client: %v", err)
	}

	// TODO: parse the buffer and send the response or error
	// TODO: This is a very basic implementation of the gateway
	// and it is synchronous. I should make it asynchronous.
	logrus.Debugf("Received %d bytes from %s", len(buf), gconn.RemoteAddr().String())

	// Send the query to the server
	err = client.Send(buf)
	if err != nil {
		return err
	}

	// Receive the response from the server
	size, response, err := client.Receive()
	if err := onOutgoingTraffic(gconn, client, response[:size], err); err != nil {
		logrus.Errorf("Error processing data from server: %s", err)
	}

	if err != nil && errors.Is(err, io.EOF) {
		// The server has closed the connection
		logrus.Error("The client is not connected to the server anymore")
		// Either the client is not connected to the server anymore or
		// server forceful closed the connection
		// Reconnect the client
		client = pr.Reconnect(client)
		// Store the client in the map, replacing the old one
		pr.clients.Store(gconn, client)
		return err
	}

	if err != nil {
		// Write the error to the client
		_, err := gconn.Write(response[:size])
		if err != nil {
			logrus.Errorf("Error writing the error to client: %v", err)
		}
		return fmt.Errorf("error receiving data from server: %w", err)
	}

	// Write the response to the incoming connection
	_, err = gconn.Write(response[:size])
	if err != nil {
		logrus.Errorf("Error writing to client: %v", err)
	}

	return nil
}

func (pr *ProxyImpl) Reconnect(cl *Client) *Client {
	// Close the client
	if cl != nil && cl.ID != "" {
		cl.Close()
	}
	return NewClient(
		pr.ClientConfig.Network,
		pr.ClientConfig.Address,
		pr.ClientConfig.ReceiveBufferSize,
	)
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
	pr.clients.Range(func(_, _ interface{}) bool {
		size++
		return true
	})

	return size
}
