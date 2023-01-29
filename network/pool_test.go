package network

import (
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/gatewayd-io/gatewayd/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestNewPool(t *testing.T) {
	cfg := logging.LoggerConfig{
		Output:     nil,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)
	pool := NewPool(logger, 0, nil, nil)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
}

func TestNewEmptyPool(t *testing.T) {
	cfg := logging.LoggerConfig{
		Output:     nil,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)
	pool := NewEmptyPool(logger)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
}

func TestPool_Put(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	cfg := logging.LoggerConfig{
		Output:     nil,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	pool := NewPool(logger, 0, nil, nil)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client1.ID, client1)
	assert.Equal(t, 1, pool.Size())
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client2.ID, client2)
	assert.Equal(t, 2, pool.Size())
}

func TestPool_Pop(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	cfg := logging.LoggerConfig{
		Output:     nil,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	pool := NewPool(logger, 0, nil, nil)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client1.ID, client1)
	assert.Equal(t, 1, pool.Size())
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client2.ID, client2)
	assert.Equal(t, 2, pool.Size())
	if c1, ok := pool.Pop(client1.ID).(*Client); !ok {
		assert.Equal(t, c1, client1)
	} else {
		assert.Equal(t, client1.ID, c1.ID)
		assert.Equal(t, 1, pool.Size())
	}
	if c2, ok := pool.Pop(client2.ID).(*Client); !ok {
		assert.Equal(t, c2, client2)
	} else {
		assert.Equal(t, client2.ID, c2.ID)
		assert.Equal(t, 0, pool.Size())
	}
}

func TestPool_Clear(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	cfg := logging.LoggerConfig{
		Output:     nil,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	pool := NewPool(logger, 0, nil, nil)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client1.ID, client1)
	assert.Equal(t, 1, pool.Size())
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client2.ID, client2)
	assert.Equal(t, 2, pool.Size())
	pool.Clear()
	assert.Equal(t, 0, pool.Size())
}

func TestPool_ForEach(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	cfg := logging.LoggerConfig{
		Output:     nil,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	pool := NewPool(logger, 0, nil, nil)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client1.ID, client1)
	assert.Equal(t, 1, pool.Size())
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client2.ID, client2)
	assert.Equal(t, 2, pool.Size())
	pool.ForEach(func(key, value interface{}) bool {
		if c, ok := value.(*Client); ok {
			assert.NotNil(t, c)
		}
		return true
	})
}

func TestPool_GetClientIDs(t *testing.T) {
	postgres := embeddedpostgres.NewDatabase()
	if err := postgres.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	cfg := logging.LoggerConfig{
		Output:     nil,
		TimeFormat: zerolog.TimeFormatUnix,
		Level:      zerolog.DebugLevel,
		NoColor:    true,
	}

	logger := logging.NewLogger(cfg)

	pool := NewPool(logger, 0, nil, nil)
	defer pool.Clear()
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.Pool())
	assert.Equal(t, 0, pool.Size())
	client1 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client1.ID, client1)
	assert.Equal(t, 1, pool.Size())
	client2 := NewClient("tcp", "localhost:5432", DefaultBufferSize, logger)
	pool.Put(client2.ID, client2)
	assert.Equal(t, 2, pool.Size())

	var ids []string
	pool.ForEach(func(key, value interface{}) bool {
		if id, ok := key.(string); ok {
			ids = append(ids, id)
		}
		return true
	})
	assert.Equal(t, 2, len(ids))
	assert.Equal(t, client1.ID, ids[0])
	assert.Equal(t, client2.ID, ids[1])
	client1.Close()
	client2.Close()
	pool.Clear()
}
