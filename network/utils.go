package network

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"syscall"

	gerr "github.com/gatewayd-io/gatewayd/errors"
	"github.com/rs/zerolog"
)

// GetRLimit returns the current system soft and hard limits for the number of open files.
func GetRLimit(logger zerolog.Logger) syscall.Rlimit {
	var limits syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limits); err != nil { //nolint:nosnakecase
		logger.Error().Err(err).Msg("Failed to get rlimit")
	}
	logger.Debug().Msgf("Current system soft limit: %d", limits.Cur)
	logger.Debug().Msgf("Current system hard limit: %d", limits.Max)
	return limits
}

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
		logger.Error().Msgf("Network %s is not supported", network)
		return "", gerr.ErrNetworkNotSupported
	}
}
