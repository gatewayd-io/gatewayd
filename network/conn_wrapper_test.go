package network

import (
	"crypto/tls"
	"net"
	"testing"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_ConnWrapper_NoTLS tests that the ConnWrapper correctly wraps a net.Conn.
func Test_ConnWrapper_NoTLS(t *testing.T) {
	server, client := net.Pipe()
	require.NotNil(t, server)
	require.NotNil(t, client)

	serverWrapper := NewConnWrapper(ConnWrapper{
		NetConn:          server,
		HandshakeTimeout: config.DefaultHandshakeTimeout,
	})
	assert.Equal(t, server, serverWrapper.Conn())
	defer serverWrapper.Close()
	assert.False(t, serverWrapper.IsTLSEnabled())
	assert.Equal(t, serverWrapper.LocalAddr(), server.LocalAddr())
	assert.Equal(t, serverWrapper.RemoteAddr(), server.RemoteAddr())

	clientWrapper := NewConnWrapper(ConnWrapper{
		NetConn:          client,
		HandshakeTimeout: config.DefaultHandshakeTimeout,
	})
	assert.Equal(t, client, clientWrapper.Conn())
	defer clientWrapper.Close()
	assert.False(t, clientWrapper.IsTLSEnabled())
	assert.Equal(t, clientWrapper.LocalAddr(), client.LocalAddr())
	assert.Equal(t, clientWrapper.RemoteAddr(), client.RemoteAddr())

	// Write and read data.
	go func() {
		sent, err := serverWrapper.Write([]byte("Hello, World!"))
		assert.Equal(t, 13, sent)
		require.NoError(t, err)
	}()

	go func() {
		greeting := make([]byte, 13)
		read, err := clientWrapper.Read(greeting)
		assert.Equal(t, 13, read)
		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", string(greeting))
	}()
}

// Test_ConnWrapper_TLS tests that the CreateTLSConfig function correctly
// creates a TLS config given a certificate and a private key.
func Test_CreateTLSConfig(t *testing.T) {
	tlsConfig, err := CreateTLSConfig(
		"../cmd/testdata/localhost.crt", "../cmd/testdata/localhost.key")
	require.NoError(t, err)
	assert.Equal(t, tlsConfig.ClientAuth, tls.VerifyClientCertIfGiven)
	assert.NotEmpty(t, tlsConfig.Certificates[0].Certificate)
	assert.NotEmpty(t, tlsConfig.Certificates[0].PrivateKey)
}
