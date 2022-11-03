package network

import (
	"sync"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRunServer(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	onIncomingTraffic := func(gconn gnet.Conn, cl *Client, buf []byte, err error) error {
		logrus.Info("Incoming traffic")
		assert.Equal(t, CreatePgStartupPacket(), buf)
		assert.Nil(t, err)
		return nil
	}

	onOutgoingTraffic := func(gconn gnet.Conn, cl *Client, buf []byte, err error) error {
		logrus.Info("Outgoing traffic")
		assert.Equal(t, CreatePostgreSQLPacket('R', []byte{0x0, 0x0, 0x0, 0x3}), buf)
		assert.Nil(t, err)
		return nil
	}

	// Create a server
	server := NewServer(
		"tcp",
		"127.0.0.1:15432",
		0,
		0,
		DefaultTickInterval,
		2,
		DefaultBufferSize,
		false,
		false,
		[]gnet.Option{
			gnet.WithMulticore(true),
		},
		onIncomingTraffic,
		onOutgoingTraffic,
	)
	assert.NotNil(t, server)

	var wg sync.WaitGroup
	wg.Add(2)

	go func(t *testing.T, server *Server) {
		t.Helper()
		defer wg.Done()

		server.Run()
	}(t, server)

	go func(t *testing.T, server *Server) {
		t.Helper()
		defer wg.Done()

		for {
			if server.IsRunning() {
				client := NewClient("tcp", "127.0.0.1:15432", DefaultBufferSize)
				defer client.Close()

				assert.NotNil(t, client)
				err := client.Send(CreatePgStartupPacket())
				assert.Nil(t, err)

				// The server should respond with a 'R' packet
				size, data, err := client.Receive()
				msg := []byte{0x0, 0x0, 0x0, 0x3}
				// This includes the message type, length and the message itself
				assert.Equal(t, 9, size)
				assert.Equal(t, len(data[:size]), size)
				assert.Nil(t, err)
				packetSize := int(data[1])<<24 | int(data[2])<<16 | int(data[3])<<8 | int(data[4])
				assert.Equal(t, 8, packetSize)
				assert.NotEmpty(t, data[:size])
				assert.Equal(t, msg, data[5:size])
				// AuthenticationOk
				assert.Equal(t, uint8(0x52), data[0])

				// Clean up
				server.Shutdown()
				assert.NoError(t, postgres.Stop())
				return
			}
		}
	}(t, server)

	wg.Wait()
}
