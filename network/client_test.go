package network

import (
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	c := NewClient("tcp", "localhost:5432", 4096)
	defer c.Close()

	assert.Equal(t, "tcp", c.Network)
	assert.Equal(t, "127.0.0.1:5432", c.Address)
	assert.Equal(t, 4096, c.ReceiveBufferSize)
	assert.NotEmpty(t, c.ID)
	assert.NotNil(t, c.Conn)
}

func TestSend(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	c := NewClient("tcp", "localhost:5432", 4096)
	defer c.Close()

	assert.NotNil(t, c)
	err := c.Send(CreatePostgreSQLPacket('Q', []byte("select 1;")))
	assert.Nil(t, err)
}

func TestReceive(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	c := NewClient("tcp", "localhost:5432", 4096)
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
	assert.Equal(t, 83, int(data[5]))
}

func TestClose(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	c := NewClient("tcp", "localhost:5432", 4096)
	assert.NotNil(t, c)
	c.Close()
	assert.Equal(t, "", c.ID)
	assert.Equal(t, "", c.Network)
	assert.Equal(t, "", c.Address)
	assert.Nil(t, c.Conn)
	assert.Equal(t, 0, c.ReceiveBufferSize)
}
