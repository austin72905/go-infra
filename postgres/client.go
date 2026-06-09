package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	pool *pgxpool.Pool
}

func New(dsn string, maxIdleConns, maxOpenConns int, connMaxIdleTime, connMaxLifetime time.Duration) *Client {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		panic(err)
	}

	if maxOpenConns > 0 {
		config.MaxConns = int32(maxOpenConns)
	}
	if maxIdleConns > 0 {
		config.MinConns = int32(maxIdleConns)
	}
	if connMaxIdleTime > 0 {
		config.MaxConnIdleTime = connMaxIdleTime
	}
	if connMaxLifetime > 0 {
		config.MaxConnLifetime = connMaxLifetime
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		panic(err)
	}

	return &Client{pool: pool}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *Client) Close() {
	c.pool.Close()
}

func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}
