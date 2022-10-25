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
// TODO: Fix the connection leak:
// unexpected EOF on client connection with an open transaction

func NewClient(network, address string, receiveBufferSize int) *Client {
	c := Client{
		Network: network,
		Address: address,
	}
	conn, err := net.Dial(c.Network, c.Address)
	if err != nil {
		logrus.Error(err)
	}
	c.Conn = conn
	if c.ReceiveBufferSize == 0 {
		c.ReceiveBufferSize = 4096
	}
	logrus.Infof("New client created: %s", c.Address)
	c.ID = GetID(conn.LocalAddr().Network(), conn.LocalAddr().String(), 1000)

	return &c
}

func (c Client) Send(data []byte) error {
	_, err := c.Write(data)
	if err != nil {
		logrus.Errorf("Couldn't send data to the server: %s", err)
		return err
	}
	logrus.Infof("Sent %d bytes to %s", len(data), c.Address)
	// logrus.Infof("Sent data: %s", data)
	return nil
}

func (c Client) Receive() (int, []byte, error) {
	buf := make([]byte, c.ReceiveBufferSize)
	read, err := c.Read(buf)
	if err != nil {
		logrus.Errorf("Couldn't receive data from the server: %s", err)
		return 0, nil, err
	}
	logrus.Infof("Received %d bytes from %s", read, c.Address)
	// logrus.Infof("Received data: %s", buf[:read])
	return read, buf, nil
}

func (c Client) Close() {
	logrus.Infof("Closing connection to %s", c.Address)
	if c.Conn != nil {
		c.Conn.Close()
	}
}
