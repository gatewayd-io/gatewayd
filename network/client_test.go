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

	client := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	defer client.Close()

	assert.Equal(t, "tcp", client.Network)
	assert.Equal(t, "127.0.0.1:5432", client.Address)
	assert.Equal(t, DefaultBufferSize, client.ReceiveBufferSize)
	assert.NotEmpty(t, client.ID)
	assert.NotNil(t, client.Conn)
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

	client := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	defer client.Close()

	assert.NotNil(t, client)
	err := client.Send(CreatePostgreSQLPacket('Q', []byte("select 1;")))
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

	client := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	defer client.Close()

	assert.NotNil(t, client)
	err := client.Send(CreatePgStartupPacket())
	assert.Nil(t, err)

	size, data, err := client.Receive()
	msg := "\x00\x00\x00\x03"
	assert.Equal(t, 9, size)
	assert.Equal(t, len(data[:size]), size)
	assert.Nil(t, err)
	assert.NotEmpty(t, data[:size])
	assert.Equal(t, msg, string(data[5:size]))
	// AuthenticationOk
	assert.Equal(t, uint8(0x52), data[0])
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

	client := NewClient("tcp", "localhost:5432", DefaultBufferSize)
	assert.NotNil(t, client)
	client.Close()
	assert.Equal(t, "", client.ID)
	assert.Equal(t, "", client.Network)
	assert.Equal(t, "", client.Address)
	assert.Nil(t, client.Conn)
	assert.Equal(t, 0, client.ReceiveBufferSize)
}
