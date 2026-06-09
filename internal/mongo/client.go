package mongo

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	orgmongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Client struct {
	uri         string
	options     *options.ClientOptions
	client      *orgmongo.Client
	initialized bool
}

func New() *Client {
	opt := options.Client()
	opt.SetMinPoolSize(10)
	opt.SetMaxPoolSize(50)
	opt.SetMaxConnecting(50)
	opt.SetMaxConnIdleTime(30 * time.Minute)
	opt.SetRetryReads(true)
	opt.SetRetryWrites(true)
	opt.SetConnectTimeout(5 * time.Second)
	opt.SetTimeout(120 * time.Second)

	return &Client{
		options: opt,
	}
}

func (c *Client) SetURI(uri string) {
	c.uri = uri
	c.options.ApplyURI(uri)
}

func (c *Client) SetReadPreference(rp *readpref.ReadPref) {
	c.options.SetReadPreference(rp)
}

func (c *Client) SetTimeout(timeout time.Duration) {
	c.options.SetTimeout(timeout)
	c.options.SetSocketTimeout(timeout)
	c.options.SetServerSelectionTimeout(3 * timeout)
}

func (c *Client) SetPoolSize(minSize, maxSize uint64) {
	c.options.SetMinPoolSize(minSize)
	c.options.SetMaxPoolSize(maxSize)
}

func (c *Client) SetMaxConnecting(maxConnecting uint64) {
	c.options.SetMaxConnecting(maxConnecting)
}

func (c *Client) SetTLSConfig(conf *tls.Config) {
	if conf != nil {
		c.options.SetTLSConfig(conf)
	}
}

func (c *Client) Initialize(ctx context.Context) error {
	if c.initialized {
		return nil
	}
	if c.uri == "" {
		return fmt.Errorf("mongo uri is required")
	}

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := orgmongo.Connect(connectCtx, c.options)
	if err != nil {
		return err
	}
	if err := client.Ping(context.Background(), nil); err != nil {
		_ = client.Disconnect(context.Background())
		return err
	}

	c.client = client
	c.initialized = true
	return nil
}

func (c *Client) Ping(ctx context.Context) error {
	if err := c.Initialize(ctx); err != nil {
		return err
	}
	return c.client.Ping(ctx, nil)
}

func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}
	err := c.client.Disconnect(context.Background())
	c.client = nil
	c.initialized = false
	return err
}

func (c *Client) Raw() *orgmongo.Client {
	if !c.initialized {
		panic("mongo client is not initialized")
	}
	return c.client
}
