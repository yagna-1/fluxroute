package coordinator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCoordinator struct {
	client redis.UniversalClient
	prefix string
}

type redisLease struct {
	client redis.UniversalClient
	key    string
	token  string
}

func NewRedisCoordinator(redisURL string, prefix string) (Coordinator, error) {
	if strings.TrimSpace(redisURL) == "" {
		return nil, fmt.Errorf("redis url is empty")
	}
	if prefix == "" {
		prefix = "agent-router"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &redisCoordinator{client: client, prefix: prefix}, nil
}

func (c *redisCoordinator) Acquire(ctx context.Context, key string, ttl time.Duration) (Lease, error) {
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	fullKey := c.prefix + ":" + key
	token, err := randomToken()
	if err != nil {
		return nil, err
	}

	for {
		ok, err := c.client.SetNX(ctx, fullKey, token, ttl).Result()
		if err != nil {
			return nil, fmt.Errorf("redis setnx failed: %w", err)
		}
		if ok {
			return &redisLease{client: c.client, key: fullKey, token: token}, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("acquire redis lease: %w", ctx.Err())
		case <-time.After(30 * time.Millisecond):
		}
	}
}

func (l *redisLease) Release(ctx context.Context) error {
	const script = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`
	if err := l.client.Eval(ctx, script, []string{l.key}, l.token).Err(); err != nil {
		return fmt.Errorf("release redis lease: %w", err)
	}
	return nil
}

func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
