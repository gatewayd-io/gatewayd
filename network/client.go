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

	return &c
}

func (c *Client) Send(data []byte) {
	_, err := c.Write(data)
	if err != nil {
		logrus.Error(err)
	}
}

func (c *Client) Receive() []byte {
	buf := make([]byte, c.ReceiveBufferSize)
	_, err := c.Read(buf)
	if err != nil {
		logrus.Error(err)
	}
	return buf
}

func (c *Client) Close() {
	c.Conn.Close()
}
