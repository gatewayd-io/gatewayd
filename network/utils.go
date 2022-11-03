package network

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"syscall"

	"github.com/sirupsen/logrus"
)

func GetRLimit() syscall.Rlimit {
	var limits syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limits); err != nil { //nolint:nosnakecase
		logrus.Error(err)
	}
	logrus.Debugf("Current system soft limit: %d", limits.Cur)
	logrus.Debugf("Current system hard limit: %d", limits.Max)
	return limits
}

func GetID(network, address string, seed int) string {
	hash := sha256.New()
	_, err := hash.Write([]byte(fmt.Sprintf("%s://%s%d", network, address, seed)))
	if err != nil {
		logrus.Error(err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func Resolve(network, address string) (string, error) {
	switch network {
	case "tcp", "tcp4", "tcp6":
		addr, err := net.ResolveTCPAddr(network, address)
		return addr.String(), err
	case "udp", "udp4", "udp6":
		addr, err := net.ResolveUDPAddr(network, address)
		return addr.String(), err
	case "unix", "unixgram", "unixpacket":
		addr, err := net.ResolveUnixAddr(network, address)
		return addr.String(), err
	default:
		logrus.Errorf("Network %s is not supported", network)
		return "", ErrNetworkNotSupported
	}
}
