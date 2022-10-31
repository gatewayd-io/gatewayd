package network

import (
	"net"

	"github.com/sirupsen/logrus"
)

type Client struct {
	net.Conn

	ID                string
	ReceiveBufferSize int
	Network           string // tcp/udp/unix
	Address           string

	// TODO: add read/write deadline and deal with timeouts
}

// TODO: implement a better connection management algorithm

func NewClient(network, address string, receiveBufferSize int) *Client {
	var client Client

	// Try to resolve the address and log an error if it can't be resolved
	addr, err := Resolve(network, address)
	if err != nil {
		logrus.Error(err)
	}

	// Create a resolved client
	client = Client{
		Network: network,
		Address: addr,
	}

	// Fall back to the original network and address if the address can't be resolved
	if client.Address == "" || client.Network == "" {
		client = Client{
			Network: network,
			Address: address,
		}
	}

	// Create a new connection
	conn, err := net.Dial(client.Network, client.Address)
	if err != nil {
		logrus.Error(err)
		return nil
	}

	client.Conn = conn
	if client.ReceiveBufferSize == 0 {
		client.ReceiveBufferSize = 4096
	}
	logrus.Debugf("New client created: %s", client.Address)
	client.ID = GetID(conn.LocalAddr().Network(), conn.LocalAddr().String(), 1000)

	return &client
}

func (c *Client) Send(data []byte) error {
	_, err := c.Write(data)
	if err != nil {
		logrus.Errorf("Couldn't send data to the server: %s", err)
		return err
	}
	logrus.Debugf("Sent %d bytes to %s", len(data), c.Address)
	// logrus.Infof("Sent data: %s", data)
	return nil
}

func (c *Client) Receive() (int, []byte, error) {
	buf := make([]byte, c.ReceiveBufferSize)
	read, err := c.Read(buf)
	if err != nil {
		logrus.Errorf("Couldn't receive data from the server: %s", err)
		return 0, nil, err
	}
	logrus.Debugf("Received %d bytes from %s", read, c.Address)
	// logrus.Infof("Received data: %s", buf[:read])
	return read, buf, nil
}

func (c *Client) Close() {
	logrus.Debugf("Closing connection to %s", c.Address)
	if c.Conn != nil {
		c.Conn.Close()
	}
	c.ID = ""
	c.Conn = nil
	c.Address = ""
	c.Network = ""
	c.ReceiveBufferSize = 0
}
