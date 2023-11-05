package network

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/rs/zerolog"
)

// GetID returns a unique ID (hash) for a network connection.
func GetID(network, address string, seed int, logger zerolog.Logger) string {
	hash := sha256.New()
	_, err := hash.Write([]byte(fmt.Sprintf("%s://%s%d", network, address, seed)))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to generate ID")
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// Resolve resolves a network address.
func Resolve(network, address string, logger zerolog.Logger) (string, *gerr.GatewayDError) {
	switch network {
	case "tcp", "tcp4", "tcp6":
		addr, err := net.ResolveTCPAddr(network, address)
		if err == nil {
			return addr.String(), nil
		}
		return "", gerr.ErrResolveFailed.Wrap(err)
	case "udp", "udp4", "udp6":
		addr, err := net.ResolveUDPAddr(network, address)
		if err == nil {
			return addr.String(), nil
		}
		return "", gerr.ErrResolveFailed.Wrap(err)
	case "unix", "unixgram", "unixpacket":
		addr, err := net.ResolveUnixAddr(network, address)
		if err == nil {
			return addr.String(), nil
		}
		return "", gerr.ErrResolveFailed.Wrap(err)
	default:
		logger.Error().Str("network", network).Msg("Network is not supported")
		return "", gerr.ErrNetworkNotSupported
	}
}

// trafficData creates the ingress/egress map for the traffic hooks.
func trafficData(
	conn net.Conn,
	client *Client,
	fields []Field,
	err interface{},
) map[string]interface{} {
	if conn == nil || client == nil {
		return nil
	}

	data := map[string]interface{}{
		"client": map[string]interface{}{
			"local":  LocalAddr(conn),
			"remote": RemoteAddr(conn),
		},
		"server": map[string]interface{}{
			"local":  client.LocalAddr(),
			"remote": client.RemoteAddr(),
		},
		"error": "",
	}

	for _, field := range fields {
		data[field.Name] = field.Value
	}

	if err != nil {
		switch typedErr := err.(type) {
		case *gerr.GatewayDError:
		case error:
			data["error"] = typedErr.Error()
		case string:
			data["error"] = typedErr
		default:
			data["error"] = fmt.Sprintf("%v", err)
		}
	}

	return data
}

// extractFieldValue extracts the given field name and error message from the result of the hook.
func extractFieldValue(result map[string]interface{}, fieldName string) ([]byte, string) {
	var data []byte
	var err string

	if result != nil {
		if val, ok := result[fieldName].([]byte); ok {
			data = val
		}

		if errMsg, ok := result["error"].(string); ok && errMsg != "" {
			err = errMsg
		}
	}
	return data, err
}

// IsConnTimedOut returns true if the error is a timeout error.
func IsConnTimedOut(err *gerr.GatewayDError) bool {
	if err != nil && err.Unwrap() != nil {
		var netErr net.Error
		if ok := errors.As(err.Unwrap(), &netErr); ok && netErr.Timeout() {
			return true
		}
	}
	return false
}

// IsConnClosed returns true if the connection is closed.
func IsConnClosed(received int, err *gerr.GatewayDError) bool {
	return received == 0 && err != nil && err.Unwrap() != nil && errors.Is(err.Unwrap(), io.EOF)
}

// LocalAddr returns the local address of the connection.
func LocalAddr(conn net.Conn) string {
	if conn != nil && conn.LocalAddr() != nil {
		return conn.LocalAddr().String()
	}
	return ""
}

// RemoteAddr returns the remote address of the connection.
func RemoteAddr(conn net.Conn) string {
	if conn != nil && conn.RemoteAddr() != nil {
		return conn.RemoteAddr().String()
	}
	return ""
}

// IsPostgresSSLRequest returns true if the message is a SSL request.
// This is copied from gatewayd-plugin-sdk to avoid the dependency on CGO.
//
//nolint:gomnd
func IsPostgresSSLRequest(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	if binary.BigEndian.Uint32(data[0:4]) != 8 {
		return false
	}

	if binary.BigEndian.Uint32(data[4:8]) != 80877103 {
		return false
	}

	return true
}
