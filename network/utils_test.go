package network

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestGetID tests the GetID function.
func TestGetID(t *testing.T) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}

	logger := logging.NewLogger(context.Background(), cfg)
	id := GetID("tcp", "localhost:5432", 1, logger)
	assert.Equal(t, "0cf47ee4e436ecb40dbd1d2d9a47179d1f6d98e2ea18d6fbd1cdfa85d3cec94f", id)
}

// TestResolve tests the Resolve function.
func TestResolve(t *testing.T) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}

	logger := logging.NewLogger(context.Background(), cfg)
	address, err := Resolve("udp", "localhost:53", logger)
	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1:53", address)
}

var seedValues = []int{1000, 10000, 100000, 1000000, 10000000}

func BenchmarkGetID(b *testing.B) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}

	logger := logging.NewLogger(context.Background(), cfg)
	for _, seed := range seedValues {
		b.Run(fmt.Sprintf("seed=%d", seed), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				GetID("tcp", "localhost:5432", seed, logger)
			}
		})
	}
}

func BenchmarkResolveUDP(b *testing.B) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}

	logger := logging.NewLogger(context.Background(), cfg)
	for i := 0; i < b.N; i++ {
		Resolve("udp", "localhost:53", logger) //nolint:errcheck
	}
}

func BenchmarkResolveTCP(b *testing.B) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}

	logger := logging.NewLogger(context.Background(), cfg)
	for i := 0; i < b.N; i++ {
		Resolve("tcp", "localhost:5432", logger) //nolint:errcheck
	}
}

func BenchmarkResolveUnix(b *testing.B) {
	cfg := logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	}

	logger := logging.NewLogger(context.Background(), cfg)
	for i := 0; i < b.N; i++ {
		Resolve("unix", "/tmp/unix.sock", logger) //nolint:errcheck
	}
}

type testConnection struct {
	*ConnWrapper
}

func (c *testConnection) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	}
}

func (c *testConnection) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	}
}

func BenchmarkTrafficData(b *testing.B) {
	logger := logging.NewLogger(context.Background(), logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	conn := &testConnection{}
	client := NewClient(context.Background(), &config.Client{
		Network:            "tcp",
		Address:            "localhost:5432",
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: time.Second * 10,
		ReceiveChunkSize:   1024,
	}, logger, nil)
	fields := []Field{
		{
			Name:  "test",
			Value: []byte("test"),
		},
		{
			Name:  "test2",
			Value: big.NewInt(123456).Bytes(),
		},
	}
	err := "test error"
	for i := 0; i < b.N; i++ {
		trafficData(conn.Conn(), client, fields, err)
	}
}

func BenchmarkExtractFieldValue(b *testing.B) {
	for i := 0; i < b.N; i++ {
		extractFieldValue(
			map[string]interface{}{
				"test": "test",
			},
			"test",
		)
	}
}
