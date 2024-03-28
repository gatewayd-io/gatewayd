//nolint:wrapcheck
package network

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	gerr "github.com/gatewayd-io/gatewayd/errors"
)

// UpgraderFunc is a function that upgrades a connection to TLS.
// For example, this function can be used to upgrade a Postgres
// connection to TLS. Postgres initially sends a SSLRequest message,
// and the server responds with a 'S' message to indicate that it
// supports TLS. The client then upgrades the connection to TLS.
// See https://www.postgresql.org/docs/current/protocol-flow.html
type UpgraderFunc func(net.Conn)

type IConnWrapper interface {
	Conn() net.Conn
	UpgradeToTLS(upgrader UpgraderFunc) *gerr.GatewayDError
	Close() error
	Write(data []byte) (int, error)
	Read(data []byte) (int, error)
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	IsTLSEnabled() bool
}

type ConnWrapper struct {
	NetConn          net.Conn
	tlsConn          *tls.Conn
	TlsConfig        *tls.Config
	isTLSEnabled     bool
	HandshakeTimeout time.Duration
}

var _ IConnWrapper = (*ConnWrapper)(nil)

// Conn returns the underlying connection.
func (cw *ConnWrapper) Conn() net.Conn {
	if cw.tlsConn != nil {
		return net.Conn(cw.tlsConn)
	}
	return cw.NetConn
}

// UpgradeToTLS upgrades the connection to TLS.
func (cw *ConnWrapper) UpgradeToTLS(upgrader UpgraderFunc) *gerr.GatewayDError {
	if cw.tlsConn != nil {
		return nil
	}

	if !cw.isTLSEnabled {
		return nil
	}

	if upgrader != nil {
		upgrader(cw.NetConn)
	}

	tlsConn := tls.Server(cw.NetConn, cw.TlsConfig)

	ctx, cancel := context.WithTimeout(context.Background(), cw.HandshakeTimeout)
	defer cancel()

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		return gerr.ErrUpgradeToTLSFailed.Wrap(err)
	}
	cw.tlsConn = tlsConn
	cw.isTLSEnabled = true
	return nil
}

// Close closes the connection.
func (cw *ConnWrapper) Close() error {
	if cw.tlsConn != nil {
		return cw.tlsConn.Close()
	}
	return cw.NetConn.Close()
}

// Write writes data to the connection.
func (cw *ConnWrapper) Write(data []byte) (int, error) {
	if cw.tlsConn != nil {
		return cw.tlsConn.Write(data)
	}
	return cw.NetConn.Write(data)
}

// Read reads data from the connection.
func (cw *ConnWrapper) Read(data []byte) (int, error) {
	if cw.tlsConn != nil {
		return cw.tlsConn.Read(data)
	}
	return cw.NetConn.Read(data)
}

// RemoteAddr returns the remote address.
func (cw *ConnWrapper) RemoteAddr() net.Addr {
	if cw.tlsConn != nil {
		return cw.tlsConn.RemoteAddr()
	}
	return cw.NetConn.RemoteAddr()
}

// LocalAddr returns the local address.
func (cw *ConnWrapper) LocalAddr() net.Addr {
	if cw.tlsConn != nil {
		return cw.tlsConn.LocalAddr()
	}
	return cw.NetConn.LocalAddr()
}

// IsTLSEnabled returns true if TLS is enabled.
func (cw *ConnWrapper) IsTLSEnabled() bool {
	return cw.tlsConn != nil || cw.isTLSEnabled
}

// NewConnWrapper creates a new connection wrapper. The connection
// wrapper is used to upgrade the connection to TLS if need be.
func NewConnWrapper(
	connWrapper ConnWrapper,
) *ConnWrapper {
	return &ConnWrapper{
		NetConn:          connWrapper.NetConn,
		TlsConfig:        connWrapper.TlsConfig,
		isTLSEnabled:     connWrapper.TlsConfig != nil && connWrapper.TlsConfig.Certificates != nil,
		HandshakeTimeout: connWrapper.HandshakeTimeout,
	}
}

// CreateTLSConfig returns a TLS config from the given cert and key.
// TODO: Make this more generic and configurable.
func CreateTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:               tls.VersionTLS13,
		Certificates:             []tls.Certificate{cert},
		ClientAuth:               tls.VerifyClientCertIfGiven,
		PreferServerCipherSuites: true,
	}, nil
}
