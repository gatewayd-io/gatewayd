package network

import (
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/assert"
)

func TestRunServer(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	server := &Server{
		Network: "tcp",
		Address: "localhost:15432",
		Options: []gnet.Option{
			gnet.WithMulticore(false),
		},
		SoftLimit: 1,
		HardLimit: 2,
	}
	assert.NotNil(t, server)

	go func(t *testing.T, server *Server) {
		server.Run()
	}(t, server)

	for {
		if server.Status == Running {
			client := NewClient("tcp", "localhost:15432", 4096)
			defer client.Close()

			assert.NotNil(t, client)
			err := client.Send(CreatePostgreSQLPacket('Q', []byte("select 1;")))
			assert.Nil(t, err)

			size, data, err := client.Receive()
			msg := "SFATAL\x00VFATAL\x00C0A000\x00Munsupported frontend protocol 0.0: server supports 3.0 to 3.0\x00Fpostmaster.c\x00L2138\x00RProcessStartupPacket\x00\x00"
			assert.Equal(t, 132, size)
			assert.Equal(t, len(data[:size]), size)
			assert.Nil(t, err)
			assert.NotEmpty(t, data[:size])
			assert.Equal(t, msg, string(data[5:size]))
			assert.Equal(t, "E", string(data[0]))

			// Clean up
			server.Shutdown()
			assert.NoError(t, postgres.Stop())
			break
		}
	}
}
