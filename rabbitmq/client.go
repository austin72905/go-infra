package rabbitmq

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	Body        []byte
	Headers     amqp.Table
	ContentType string
	Exchange    string
	RoutingKey  string
	DeliveryTag uint64
	Redelivered bool
}

type Handler func(ctx context.Context, message Message) error

type PublishOptions struct {
	ContentType string
	Headers     amqp.Table
	Persistent  bool
	Mandatory   bool
	Immediate   bool
}

type PublishOption func(*PublishOptions)

func WithContentType(contentType string) PublishOption {
	return func(opts *PublishOptions) {
		opts.ContentType = contentType
	}
}

func WithHeaders(headers amqp.Table) PublishOption {
	return func(opts *PublishOptions) {
		opts.Headers = headers
	}
}

func WithPersistentDelivery() PublishOption {
	return func(opts *PublishOptions) {
		opts.Persistent = true
	}
}

func WithMandatoryPublish() PublishOption {
	return func(opts *PublishOptions) {
		opts.Mandatory = true
	}
}

type Client struct {
	url string

	mu            sync.Mutex
	publishMu     sync.Mutex
	conn          *amqp.Connection
	publishCh     *amqp.Channel
	publishChOpen bool
}

func New(url string) *Client {
	return &Client{url: url}
}

func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && !c.conn.IsClosed() {
		return nil
	}

	conn, err := amqp.Dial(c.url)
	if err != nil {
		return fmt.Errorf("connect rabbitmq failed: %w", err)
	}

	c.conn = conn
	c.publishCh = nil
	c.publishChOpen = false
	return nil
}

func (c *Client) Channel() (*amqp.Channel, error) {
	conn, err := c.connection()
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open rabbitmq channel failed: %w", err)
	}
	return ch, nil
}

func (c *Client) Publish(ctx context.Context, exchange, routingKey string, body []byte, options ...PublishOption) error {
	if routingKey == "" {
		return fmt.Errorf("rabbitmq routing key is required")
	}

	opts := PublishOptions{}
	for _, option := range options {
		if option != nil {
			option(&opts)
		}
	}

	deliveryMode := uint8(amqp.Transient)
	if opts.Persistent {
		deliveryMode = amqp.Persistent
	}

	c.publishMu.Lock()
	defer c.publishMu.Unlock()

	ch, err := c.publisherChannel()
	if err != nil {
		return err
	}

	err = ch.PublishWithContext(ctx, exchange, routingKey, opts.Mandatory, opts.Immediate, amqp.Publishing{
		Headers:      opts.Headers,
		ContentType:  opts.ContentType,
		Body:         append([]byte(nil), body...),
		DeliveryMode: deliveryMode,
		Timestamp:    time.Now(),
	})
	if err != nil {
		return fmt.Errorf("publish rabbitmq message failed: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var firstErr error
	if c.publishCh != nil {
		if err := c.publishCh.Close(); err != nil {
			firstErr = err
		}
		c.publishCh = nil
		c.publishChOpen = false
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		c.conn = nil
	}
	return firstErr
}

func (c *Client) connection() (*amqp.Connection, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}
	return c.conn, nil
}

func (c *Client) publisherChannel() (*amqp.Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || c.conn.IsClosed() {
		conn, err := amqp.Dial(c.url)
		if err != nil {
			return nil, fmt.Errorf("connect rabbitmq failed: %w", err)
		}
		c.conn = conn
		c.publishCh = nil
		c.publishChOpen = false
	}

	if c.publishCh != nil && c.publishChOpen {
		return c.publishCh, nil
	}

	ch, err := c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open rabbitmq publisher channel failed: %w", err)
	}

	c.publishCh = ch
	c.publishChOpen = true
	go func() {
		_, ok := <-ch.NotifyClose(make(chan *amqp.Error, 1))
		if ok {
			c.mu.Lock()
			if c.publishCh == ch {
				c.publishChOpen = false
			}
			c.mu.Unlock()
		}
	}()

	return c.publishCh, nil
}

type ConsumerOptions struct {
	ConsumerTag string
	AutoAck     bool
	Prefetch    int
	Requeue     bool
	Args        amqp.Table
}

type ConsumerRunner struct {
	client  *Client
	queue   string
	handler Handler
	options ConsumerOptions

	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	ch           *amqp.Channel
	runningTasks atomic.Int32
	started      atomic.Bool
}

func NewConsumerRunner(client *Client, queue string, handler Handler, options ConsumerOptions) *ConsumerRunner {
	if client == nil {
		panic("rabbitmq client is required")
	}
	if queue == "" {
		panic("rabbitmq queue is required")
	}
	if handler == nil {
		panic("rabbitmq handler is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &ConsumerRunner{
		client:  client,
		queue:   queue,
		handler: handler,
		options: options,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (r *ConsumerRunner) Start() {
	if !r.started.CompareAndSwap(false, true) {
		return
	}

	ch, err := r.client.Channel()
	if err != nil {
		panic(fmt.Sprintf("open rabbitmq consumer channel failed: %v", err))
	}
	r.ch = ch

	if r.options.Prefetch > 0 {
		if err := ch.Qos(r.options.Prefetch, 0, false); err != nil {
			panic(fmt.Sprintf("set rabbitmq qos failed: %v", err))
		}
	}

	deliveries, err := ch.Consume(
		r.queue,
		r.options.ConsumerTag,
		r.options.AutoAck,
		false,
		false,
		false,
		r.options.Args,
	)
	if err != nil {
		panic(fmt.Sprintf("consume rabbitmq queue %q failed: %v", r.queue, err))
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		for {
			select {
			case <-r.ctx.Done():
				return
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}
				r.handle(delivery)
			}
		}
	}()
}

func (r *ConsumerRunner) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	if r.ch != nil {
		if r.options.ConsumerTag != "" {
			_ = r.ch.Cancel(r.options.ConsumerTag, false)
		}
		_ = r.ch.Close()
	}
}

func (r *ConsumerRunner) AwaitTermination(ctx context.Context) {
	for {
		if r.runningTasks.Load() == 0 {
			break
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return
	case <-done:
		return
	case <-time.After(30 * time.Second):
		return
	}
}

func (r *ConsumerRunner) RunningTasks() int {
	return int(r.runningTasks.Load())
}

func (r *ConsumerRunner) handle(delivery amqp.Delivery) {
	r.runningTasks.Add(1)
	defer r.runningTasks.Add(-1)

	err := r.handler(r.ctx, Message{
		Body:        append([]byte(nil), delivery.Body...),
		Headers:     delivery.Headers,
		ContentType: delivery.ContentType,
		Exchange:    delivery.Exchange,
		RoutingKey:  delivery.RoutingKey,
		DeliveryTag: delivery.DeliveryTag,
		Redelivered: delivery.Redelivered,
	})
	if r.options.AutoAck {
		return
	}
	if err == nil {
		_ = delivery.Ack(false)
		return
	}
	_ = delivery.Nack(false, r.options.Requeue)
}
