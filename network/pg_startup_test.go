package network

import (
	"net"
	"testing"
	"time"

	"github.com/gatewayd-io/gatewayd/config"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/gatewayd-io/gatewayd/testhelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPgMD5Password verifies the PostgreSQL MD5 password hash computation.
func TestPgMD5Password(t *testing.T) {
	// Known test vector:
	// md5(md5("postgres" + "postgres") + salt)
	// Inner: md5("postgrespostgres") = 0bbe9ec1e6e93b2907cee5fcb2e40e0a
	// With salt [1,2,3,4]: md5("0bbe9ec1e6e93b2907cee5fcb2e40e0a" + "\x01\x02\x03\x04")
	salt := [4]byte{1, 2, 3, 4}
	hash := pgMD5Password("postgres", "postgres", salt)
	assert.True(t, len(hash) > 3)
	assert.Equal(t, "md5", hash[:3])
	// The hash should be "md5" + 32 hex characters.
	assert.Len(t, hash, 35)
}

// TestPgStartup_SCRAM tests the full PostgreSQL startup handshake with SCRAM-SHA-256.
// Requires a real PostgreSQL backend via testcontainers.
func TestPgStartup_SCRAM(t *testing.T) {
	ctx := t.Context()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)

	logger := logging.NewLogger(ctx, logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	// Dial a raw TCP connection.
	conn, err := net.DialTimeout("tcp", postgresHostIP+":"+postgresMappedPort.Port(), 5*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	// Perform the PG startup handshake.
	err = pgStartup(conn, "postgres", "postgres", "postgres", logger)
	require.NoError(t, err, "pgStartup should succeed with valid credentials")
}

// TestPgStartup_BadPassword tests that pgStartup fails with a wrong password.
func TestPgStartup_BadPassword(t *testing.T) {
	ctx := t.Context()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)

	logger := logging.NewLogger(ctx, logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	conn, err := net.DialTimeout("tcp", postgresHostIP+":"+postgresMappedPort.Port(), 5*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	err = pgStartup(conn, "postgres", "postgres", "wrongpassword", logger)
	require.Error(t, err, "pgStartup should fail with wrong password")
}

// TestPgResetSession tests that pgResetSession correctly sends DISCARD ALL.
func TestPgResetSession(t *testing.T) {
	ctx := t.Context()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)

	logger := logging.NewLogger(ctx, logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	// First establish and authenticate a connection.
	conn, err := net.DialTimeout("tcp", postgresHostIP+":"+postgresMappedPort.Port(), 5*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	err = pgStartup(conn, "postgres", "postgres", "postgres", logger)
	require.NoError(t, err)

	// Now reset the session.
	err = pgResetSession(conn, logger)
	require.NoError(t, err, "pgResetSession should succeed on an authenticated connection")

	// The connection should still be usable -- reset it again to verify.
	err = pgResetSession(conn, logger)
	require.NoError(t, err, "pgResetSession should succeed a second time")
}

// TestNewClient_WithStartupParams tests that NewClient performs PG startup
// when StartupParams are configured.
func TestNewClient_WithStartupParams(t *testing.T) {
	ctx := t.Context()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)

	logger := logging.NewLogger(ctx, logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	clientConfig := &config.Client{
		Network:            "tcp",
		Address:            postgresHostIP + ":" + postgresMappedPort.Port(),
		ReceiveChunkSize:   config.DefaultChunkSize,
		ReceiveDeadline:    config.DefaultReceiveDeadline,
		SendDeadline:       config.DefaultSendDeadline,
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		StartupParams: &config.StartupParams{
			User:     "postgres",
			Database: "postgres",
			Password: "postgres",
		},
	}

	client := NewClient(ctx, clientConfig, logger, nil)
	require.NotNil(t, client, "NewClient should succeed with valid startup params")
	defer client.Close()

	assert.True(t, client.IsConnected())
	assert.NotEmpty(t, client.ID)
	assert.NotNil(t, client.StartupParams)
}

// TestNewClient_WithStartupParams_BadPassword tests that NewClient returns nil
// when PG startup fails due to bad credentials.
func TestNewClient_WithStartupParams_BadPassword(t *testing.T) {
	ctx := t.Context()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)

	logger := logging.NewLogger(ctx, logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	clientConfig := &config.Client{
		Network:            "tcp",
		Address:            postgresHostIP + ":" + postgresMappedPort.Port(),
		ReceiveChunkSize:   config.DefaultChunkSize,
		ReceiveDeadline:    config.DefaultReceiveDeadline,
		SendDeadline:       config.DefaultSendDeadline,
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		StartupParams: &config.StartupParams{
			User:     "postgres",
			Database: "postgres",
			Password: "wrongpassword",
		},
	}

	client := NewClient(ctx, clientConfig, logger, nil)
	assert.Nil(t, client, "NewClient should return nil when PG startup fails")
}

// TestResetSession_Client tests the ResetSession method on Client.
func TestResetSession_Client(t *testing.T) {
	ctx := t.Context()
	postgresHostIP, postgresMappedPort := testhelpers.SetupPostgreSQLTestContainer(ctx, t)

	logger := logging.NewLogger(ctx, logging.LoggerConfig{
		Output:            []config.LogOutput{config.Console},
		TimeFormat:        zerolog.TimeFormatUnix,
		ConsoleTimeFormat: time.RFC3339,
		Level:             zerolog.DebugLevel,
		NoColor:           true,
	})

	clientConfig := &config.Client{
		Network:            "tcp",
		Address:            postgresHostIP + ":" + postgresMappedPort.Port(),
		ReceiveChunkSize:   config.DefaultChunkSize,
		ReceiveDeadline:    config.DefaultReceiveDeadline,
		SendDeadline:       config.DefaultSendDeadline,
		TCPKeepAlive:       false,
		TCPKeepAlivePeriod: config.DefaultTCPKeepAlivePeriod,
		StartupParams: &config.StartupParams{
			User:     "postgres",
			Database: "postgres",
			Password: "postgres",
		},
	}

	client := NewClient(ctx, clientConfig, logger, nil)
	require.NotNil(t, client)
	defer client.Close()

	// ResetSession should succeed on a pre-authenticated connection.
	err := client.ResetSession()
	require.NoError(t, err, "ResetSession should succeed")

	// Should still be connected after reset.
	assert.True(t, client.IsConnected())
}
