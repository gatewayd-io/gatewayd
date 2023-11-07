//nolint:wrapcheck
package network

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	gerr "github.com/gatewayd-io/gatewayd/errors"
)

// UpgraderFunc is a function that upgrades a connection to TLS.
// For example, this function can be used to upgrade a Postgres
// connection to TLS. Postgres initially sends a SSLRequest message,
// and the server responds with a 'S' message to indicate that it
// supports TLS. The client then upgrades the connection to TLS.
// See https://www.postgresql.org/docs/current/protocol-flow.html
type UpgraderFunc func(net.Conn) error

//nolint:interfacebloat
type IConnWrapper interface {
	Conn() net.Conn
	UpgradeToTLS(connType config.ConnectionType, upgrader UpgraderFunc) *gerr.GatewayDError
	Close() error
	Write(data []byte) (int, error)
	Read(data []byte) (int, error)
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	IsTLSEnabled() bool
	TLSConfig() *tls.Config
}

type ConnWrapper struct {
	netConn          net.Conn
	tlsConn          *tls.Conn
	tlsConfig        *tls.Config
	isTLSEnabled     bool
	handshakeTimeout time.Duration
}

var _ IConnWrapper = (*ConnWrapper)(nil)

// Conn returns the underlying connection.
func (cw *ConnWrapper) Conn() net.Conn {
	if cw.tlsConn != nil {
		return net.Conn(cw.tlsConn)
	}
	return cw.netConn
}

// UpgradeToTLS upgrades the connection to TLS.
func (cw *ConnWrapper) UpgradeToTLS(connType config.ConnectionType, upgrader UpgraderFunc) *gerr.GatewayDError {
	if cw.tlsConn != nil {
		return nil
	}

	if !cw.isTLSEnabled {
		return nil
	}

	if upgrader != nil {
		if err := upgrader(cw.netConn); err != nil {
			return gerr.ErrUpgradeToTLSFailed.Wrap(err)
		}
	}

	var tlsConn *tls.Conn
	if connType == config.ConnectionTypeClient {
		tlsConn = tls.Client(cw.netConn, cw.tlsConfig)
	} else {
		tlsConn = tls.Server(cw.netConn, cw.tlsConfig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cw.handshakeTimeout)
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
	return cw.netConn.Close()
}

// Write writes data to the connection.
func (cw *ConnWrapper) Write(data []byte) (int, error) {
	if cw.tlsConn != nil {
		return cw.tlsConn.Write(data)
	}
	return cw.netConn.Write(data)
}

// Read reads data from the connection.
func (cw *ConnWrapper) Read(data []byte) (int, error) {
	if cw.tlsConn != nil {
		return cw.tlsConn.Read(data)
	}
	return cw.netConn.Read(data)
}

// SetDeadline sets the deadline for the connection.
func (cw *ConnWrapper) SetDeadline(t time.Time) error {
	if cw.tlsConn != nil {
		return cw.tlsConn.SetDeadline(t)
	}
	return cw.netConn.SetDeadline(t)
}

// SetReadDeadline sets the read deadline for the connection.
func (cw *ConnWrapper) SetReadDeadline(t time.Time) error {
	if cw.tlsConn != nil {
		return cw.tlsConn.SetReadDeadline(t)
	}
	return cw.netConn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline for the connection.
func (cw *ConnWrapper) SetWriteDeadline(t time.Time) error {
	if cw.tlsConn != nil {
		return cw.tlsConn.SetWriteDeadline(t)
	}
	return cw.netConn.SetWriteDeadline(t)
}

// RemoteAddr returns the remote address.
func (cw *ConnWrapper) RemoteAddr() net.Addr {
	if cw.tlsConn != nil {
		return cw.tlsConn.RemoteAddr()
	}
	return cw.netConn.RemoteAddr()
}

// LocalAddr returns the local address.
func (cw *ConnWrapper) LocalAddr() net.Addr {
	if cw.tlsConn != nil {
		return cw.tlsConn.LocalAddr()
	}
	return cw.netConn.LocalAddr()
}

// IsTLSEnabled returns true if TLS is enabled.
func (cw *ConnWrapper) IsTLSEnabled() bool {
	return cw.tlsConn != nil || cw.isTLSEnabled
}

// TLSConfig returns the TLS config.
func (cw *ConnWrapper) TLSConfig() *tls.Config {
	return cw.tlsConfig
}

// NewConnWrapper creates a new connection wrapper. The connection
// wrapper is used to upgrade the connection to TLS if need be.
func NewConnWrapper(
	conn net.Conn, tlsConfig *tls.Config, handshakeTimeout time.Duration,
) *ConnWrapper {
	return &ConnWrapper{
		netConn:          conn,
		tlsConfig:        tlsConfig,
		isTLSEnabled:     tlsConfig != nil && tlsConfig.Certificates != nil,
		handshakeTimeout: handshakeTimeout,
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
		InsecureSkipVerify:       true,
	}, nil
}
