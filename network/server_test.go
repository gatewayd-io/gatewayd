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

	s := &Server{
		Network: "tcp",
		Address: "localhost:15432",
		Options: []gnet.Option{
			gnet.WithMulticore(false),
		},
		SoftLimit: 1,
		HardLimit: 2,
	}
	assert.NotNil(t, s)

	go func(t *testing.T, s *Server) {
		s.Run()
	}(t, s)

	for {
		if s.Status == Running {
			c := NewClient("tcp", "localhost:15432", 4096)
			defer c.Close()

			assert.NotNil(t, c)
			err := c.Send(CreatePostgreSQLPacket('Q', []byte("select 1;")))
			assert.Nil(t, err)

			size, data, err := c.Receive()
			msg := "SFATAL\x00VFATAL\x00C0A000\x00Munsupported frontend protocol 0.0: server supports 3.0 to 3.0\x00Fpostmaster.c\x00L2138\x00RProcessStartupPacket\x00\x00"
			assert.Equal(t, 132, size)
			assert.Equal(t, len(data[:size]), size)
			assert.Nil(t, err)
			assert.NotEmpty(t, data[:size])
			assert.Equal(t, msg, string(data[5:size]))
			assert.Equal(t, "E", string(data[0]))

			// Clean up
			s.Shutdown()
			assert.NoError(t, postgres.Stop())
			break
		}
	}
}
