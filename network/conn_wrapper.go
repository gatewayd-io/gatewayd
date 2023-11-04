package network

import (
	"crypto/tls"
	"net"
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
	LoadTLSConfig(cert, key []byte) *tls.Config
	UpgradeToTLS(upgrader UpgraderFunc) error
	Close() error
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
}

type ConnWrapper struct {
	netConn   net.Conn
	tlsConn   *tls.Conn
	tlsConfig *tls.Config
}

var _ IConnWrapper = &ConnWrapper{}

// Conn returns the underlying connection.
func (cw *ConnWrapper) Conn() net.Conn {
	if cw.tlsConn != nil {
		return net.Conn(cw.tlsConn)
	}
	return cw.netConn
}

// UpgradeToTLS upgrades the connection to TLS.
func (cw *ConnWrapper) UpgradeToTLS(upgrader UpgraderFunc) error {
	if cw.tlsConn != nil {
		return nil
	}

	if upgrader != nil {
		upgrader(cw.netConn)
	}

	tlsConn := tls.Server(cw.netConn, cw.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return err
	}
	cw.tlsConn = tlsConn
	return nil
}

// LoadTLSConfig loads the TLS config.
// TODO: Add support for client authentication.
// TODO: Should it even be here?
func (cw *ConnWrapper) LoadTLSConfig(cert, key []byte) *tls.Config {
	certPair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil
	}
	cw.tlsConfig.Certificates = []tls.Certificate{certPair}
	return cw.tlsConfig
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

// NewConnWrapper creates a new connection wrapper. The connection
// wrapper is used to upgrade the connection to TLS if need be.
func NewConnWrapper(conn net.Conn, tlsConfig *tls.Config) (*ConnWrapper, error) {
	if tlsConfig == nil {
		// TODO: Make this configurable.
		cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
		if err != nil {
			return nil, err
		}

		tlsConfig = &tls.Config{
			MinVersion:               tls.VersionTLS13,
			Certificates:             []tls.Certificate{cert},
			ClientAuth:               tls.VerifyClientCertIfGiven,
			PreferServerCipherSuites: true,
		}
	}

	return &ConnWrapper{
		netConn:   conn,
		tlsConfig: tlsConfig,
	}, nil
}
