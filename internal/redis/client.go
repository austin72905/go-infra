package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	inner *goredis.Client
}

func New(addr string, db int, password string) *Client {
	return &Client{
		inner: goredis.NewClient(&goredis.Options{
			Addr:     addr,
			DB:       db,
			Password: password,
		}),
	}
}

// 確認 Redis 連線是不是活的。
func (c *Client) Ping(ctx context.Context) error {
	return c.inner.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.inner.Close()
}

func (c *Client) Raw() *goredis.Client {
	return c.inner
}
