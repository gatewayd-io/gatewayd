package network

import (
	"net"

	"github.com/sirupsen/logrus"
)

type Client struct {
	net.Conn

	ReceiveBufferSize int
	Network           string // tcp/udp/unix
	Address           string
}

func NewClient(network, address string, receiveBufferSize int) *Client {
	c := Client{Network: network, Address: address}
	conn, err := net.Dial(c.Network, c.Address)
	if err != nil {
		logrus.Error(err)
	}
	c.Conn = conn
	if c.ReceiveBufferSize == 0 {
		c.ReceiveBufferSize = 4096
	}
	logrus.Infof("New client created: %s", c.Address)

	return &c
}

func (c *Client) Send(data []byte) {
	_, err := c.Write(data)
	if err != nil {
		logrus.Error(err)
	}
	logrus.Infof("Sent %d bytes to %s", len(data), c.Address)
}

func (c *Client) Receive() []byte {
	buf := make([]byte, c.ReceiveBufferSize)
	read, err := c.Read(buf)
	if err != nil {
		logrus.Error(err)
	}
	logrus.Infof("Received %d bytes from %s", read, c.Address)
	return buf
}

func (c *Client) Close() {
	c.Conn.Close()
	logrus.Infof("Closed connection to %s", c.Address)
}
