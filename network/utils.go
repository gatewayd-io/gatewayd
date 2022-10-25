package network

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"syscall"

	"github.com/sirupsen/logrus"
)

func GetRLimit() syscall.Rlimit {
	var limits syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limits); err != nil {
		logrus.Error(err)
	}
	logrus.Infof("Current system soft limit: %d", limits.Cur)
	logrus.Infof("Current system hard limit: %d", limits.Max)
	return limits
}

func GetID(network, address string, seed int) string {
	hash := sha1.New()
	_, err := hash.Write([]byte(fmt.Sprintf("%s://%s%d", network, address, seed)))
	if err != nil {
		logrus.Error(err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}
