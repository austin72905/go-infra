package db

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Client struct {
	sqlDB  *sql.DB
	gormDB *gorm.DB
}

func New(dsn string, maxIdleConns, maxOpenConns int, connMaxIdleTime time.Duration) *Client {
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		panic(err)
	}

	if maxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(maxIdleConns)
	}
	if maxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(maxOpenConns)
	}
	if connMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
	}

	return &Client{
		sqlDB:  sqlDB,
		gormDB: gormDB,
	}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.sqlDB.PingContext(ctx)
}

func (c *Client) Close() error {
	return c.sqlDB.Close()
}

func (c *Client) SQLDB() *sql.DB {
	return c.sqlDB
}

func (c *Client) GormDB() *gorm.DB {
	return c.gormDB
}
