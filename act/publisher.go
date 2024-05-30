package act

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type IPublisher interface {
	Publish(ctx context.Context, payload []byte) error
}

var _ IPublisher = (*Publisher)(nil)

type Publisher struct {
	Logger      zerolog.Logger
	RedisDB     redis.Cmdable
	ChannelName string
}

func NewPublisher(publisher Publisher) (*Publisher, error) {
	if err := publisher.RedisDB.Ping(context.Background()).Err(); err != nil {
		publisher.Logger.Err(err).Msg("failed to connect redis")
	}
	return &Publisher{
		Logger:      publisher.Logger,
		RedisDB:     publisher.RedisDB,
		ChannelName: publisher.ChannelName,
	}, nil
}

func (p *Publisher) Publish(ctx context.Context, payload []byte) error {
	if err := p.RedisDB.Publish(ctx, p.ChannelName, payload).Err(); err != nil {
		p.Logger.Err(err).Str("ChannelName", p.ChannelName).Msg("failed to publish task to redis")
		return fmt.Errorf("failed to publish task to redis: %w", err)
	}
	return nil
}
