package redis

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"time"
)

type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type Client struct {
	client *redis.Client
	logger *zap.Logger
}

func NewClient(cfg Config, logger *zap.Logger) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Client{
		client: client,
		logger: logger,
	}, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	tr := otel.Tracer("redis")
	ctx, span := tr.Start(ctx, "redis.get")
	defer span.End()

	span.SetAttributes(
		attribute.String("redis.key", key),
		attribute.String("redis.operation", "GET"),
	)

	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err != redis.Nil {
			c.logger.Error("failed to get key from redis",
				zap.String("key", key),
				zap.Error(err),
			)
		}
		return "", fmt.Errorf("failed to get key from redis: %w", err)
	}

	return val, nil
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	tr := otel.Tracer("redis")
	ctx, span := tr.Start(ctx, "redis.set")
	defer span.End()

	span.SetAttributes(
		attribute.String("redis.key", key),
		attribute.String("redis.operation", "SET"),
	)

	err := c.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		c.logger.Error("failed to set key in redis",
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("failed to set key in redis: %w", err)
	}

	return nil
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	tr := otel.Tracer("redis")
	ctx, span := tr.Start(ctx, "redis.del")
	defer span.End()

	span.SetAttributes(
		attribute.StringSlice("redis.keys", keys),
		attribute.String("redis.operation", "DEL"),
	)

	err := c.client.Del(ctx, keys...).Err()
	if err != nil {
		c.logger.Error("failed to delete keys from redis",
			zap.Strings("keys", keys),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete keys from redis: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	return c.client.Close()
}
