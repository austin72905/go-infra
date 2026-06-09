package rabbitmq

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/austin72905/go-infra/app"
	internalmodule "github.com/austin72905/go-infra/internal/module"
)

const DefaultComponentName = "RabbitMQ"

type Component struct {
	appRuntime *app.Runtime

	url         string
	exchange    string
	queue       string
	routingKey  string
	consumerTag string
	prefetch    int
	autoAck     bool
	requeue     bool

	client            *Client
	consumerRunners   []*ConsumerRunner
	startRegistered   bool
	stopRegistered    bool
	releaseRegistered bool
}

func (rc *Component) Initialize(runtime *app.Runtime, name string) {
	rc.appRuntime = runtime

	if !rc.startRegistered {
		runtime.Lifecycle.Startup.AddStart(internalmodule.TaskFunc(func(ctx context.Context) {
			rc.Start()
		}))
		rc.startRegistered = true
	}

	if !rc.stopRegistered {
		runtime.Lifecycle.Shutdown.AddStopBackground(internalmodule.TaskFunc(func(ctx context.Context) {
			rc.Stop()
		}))
		runtime.Lifecycle.Shutdown.AddAwaitBackground(internalmodule.TaskFunc(func(ctx context.Context) {
			rc.AwaitTermination(ctx)
		}))
		rc.stopRegistered = true
	}

	if !rc.releaseRegistered {
		runtime.Lifecycle.Shutdown.AddReleaseResources(internalmodule.TaskFunc(func(ctx context.Context) {
			_ = rc.Close()
		}))
		rc.releaseRegistered = true
	}
}

func (rc *Component) Validate() {
}

func (rc *Component) SetURL(urlValue string) {
	rc.url = urlValue
	if hostPort := hostPortFromURL(urlValue); hostPort != "" {
		rc.appRuntime.Probe.AddHostURI(hostPort)
	}
}

func (rc *Component) SetExchange(exchange string) {
	rc.exchange = exchange
}

func (rc *Component) SetQueue(queue string) {
	rc.queue = queue
}

func (rc *Component) SetRoutingKey(routingKey string) {
	rc.routingKey = routingKey
}

func (rc *Component) SetConsumerTag(consumerTag string) {
	rc.consumerTag = consumerTag
}

func (rc *Component) SetPrefetch(prefetch int) {
	if prefetch < 0 {
		prefetch = 0
	}
	rc.prefetch = prefetch
}

func (rc *Component) SetAutoAck(autoAck bool) {
	rc.autoAck = autoAck
}

func (rc *Component) SetRequeue(requeue bool) {
	rc.requeue = requeue
}

func (rc *Component) LoadFromPrefix(prefix string) {
	if prefix == "" {
		panic("rabbitmq property prefix is required")
	}

	rc.SetURL(rc.appRuntime.Property.RequiredProperty(prefix + ".url"))
	rc.SetExchange(rc.appRuntime.Property.Property(prefix + ".exchange"))
	rc.SetQueue(rc.appRuntime.Property.Property(prefix + ".queue"))
	rc.SetRoutingKey(rc.appRuntime.Property.Property(prefix + ".routing.key"))
	rc.SetConsumerTag(rc.appRuntime.Property.Property(prefix + ".consumer.tag"))

	prefetchValue := rc.appRuntime.Property.Property(prefix + ".prefetch")
	if prefetchValue != "" {
		prefetch, err := strconv.Atoi(prefetchValue)
		if err != nil {
			panic(fmt.Sprintf("invalid rabbitmq prefetch for prefix %q: %v", prefix, err))
		}
		rc.SetPrefetch(prefetch)
	}

	autoAckValue := rc.appRuntime.Property.Property(prefix + ".auto.ack")
	if autoAckValue != "" {
		autoAck, err := strconv.ParseBool(autoAckValue)
		if err != nil {
			panic(fmt.Sprintf("invalid rabbitmq auto ack for prefix %q: %v", prefix, err))
		}
		rc.SetAutoAck(autoAck)
	}

	requeueValue := rc.appRuntime.Property.Property(prefix + ".requeue")
	if requeueValue != "" {
		requeue, err := strconv.ParseBool(requeueValue)
		if err != nil {
			panic(fmt.Sprintf("invalid rabbitmq requeue for prefix %q: %v", prefix, err))
		}
		rc.SetRequeue(requeue)
	}
}

func (rc *Component) Client() *Client {
	return rc.ensureClient()
}

func (rc *Component) Publish(ctx context.Context, body []byte, options ...PublishOption) error {
	return rc.PublishTo(ctx, rc.exchange, rc.routingKey, body, options...)
}

func (rc *Component) PublishTo(ctx context.Context, exchange, routingKey string, body []byte, options ...PublishOption) error {
	if routingKey == "" {
		return fmt.Errorf("rabbitmq routing key is required")
	}
	return rc.ensureClient().Publish(ctx, exchange, routingKey, body, options...)
}

func (rc *Component) Subscribe(handler Handler) {
	rc.SubscribeQueue(rc.queue, handler)
}

func (rc *Component) SubscribeQueue(queue string, handler Handler) {
	if queue == "" {
		panic("rabbitmq queue is required")
	}
	if handler == nil {
		panic("rabbitmq handler is required")
	}

	runner := NewConsumerRunner(rc.ensureClient(), queue, handler, ConsumerOptions{
		ConsumerTag: rc.consumerTag,
		AutoAck:     rc.autoAck,
		Prefetch:    rc.prefetch,
		Requeue:     rc.requeue,
	})
	rc.consumerRunners = append(rc.consumerRunners, runner)
}

func (rc *Component) Start() {
	if len(rc.consumerRunners) == 0 {
		return
	}
	if err := rc.ensureClient().Connect(); err != nil {
		panic(err)
	}
	for _, runner := range rc.consumerRunners {
		runner.Start()
	}
}

func (rc *Component) Stop() {
	for _, runner := range rc.consumerRunners {
		runner.Stop()
	}
}

func (rc *Component) AwaitTermination(ctx context.Context) {
	for _, runner := range rc.consumerRunners {
		runner.AwaitTermination(ctx)
	}
}

func (rc *Component) RunningTasks() int {
	total := 0
	for _, runner := range rc.consumerRunners {
		total += runner.RunningTasks()
	}
	return total
}

func (rc *Component) Close() error {
	if rc.client == nil {
		return nil
	}
	err := rc.client.Close()
	rc.client = nil
	return err
}

func (rc *Component) ensureClient() *Client {
	if rc.client != nil {
		return rc.client
	}
	if rc.url == "" {
		panic("rabbitmq url is required")
	}
	rc.client = New(rc.url)
	return rc.client
}

func ComponentName(name string) string {
	if name == "" {
		return DefaultComponentName
	}
	return DefaultComponentName + ":" + name
}

func hostPortFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return ""
	}
	host := parsed.Hostname()
	port := parsed.Port()
	if host == "" {
		return ""
	}
	if port == "" {
		port = "5672"
	}
	return net.JoinHostPort(host, port)
}
