package module

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	internalmodule "go-infra/internal/module"
	"go-infra/rabbitmq"
)

const defaultRabbitMQComponentName = "RabbitMQ"

type RabbitMQComponent struct {
	appRuntime *AppRuntime

	url         string
	exchange    string
	queue       string
	routingKey  string
	consumerTag string
	prefetch    int
	autoAck     bool
	requeue     bool

	client            *rabbitmq.Client
	consumerRunners   []*rabbitmq.ConsumerRunner
	startRegistered   bool
	stopRegistered    bool
	releaseRegistered bool
}

func (rc *RabbitMQComponent) initialize(runtime *AppRuntime, name string) {
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

func (rc *RabbitMQComponent) validate() {
}

func (rc *RabbitMQComponent) SetURL(urlValue string) {
	rc.url = urlValue
	if hostPort := rabbitMQHostPort(urlValue); hostPort != "" {
		rc.appRuntime.Probe.AddHostURI(hostPort)
	}
}

func (rc *RabbitMQComponent) SetExchange(exchange string) {
	rc.exchange = exchange
}

func (rc *RabbitMQComponent) SetQueue(queue string) {
	rc.queue = queue
}

func (rc *RabbitMQComponent) SetRoutingKey(routingKey string) {
	rc.routingKey = routingKey
}

func (rc *RabbitMQComponent) SetConsumerTag(consumerTag string) {
	rc.consumerTag = consumerTag
}

func (rc *RabbitMQComponent) SetPrefetch(prefetch int) {
	if prefetch < 0 {
		prefetch = 0
	}
	rc.prefetch = prefetch
}

func (rc *RabbitMQComponent) SetAutoAck(autoAck bool) {
	rc.autoAck = autoAck
}

func (rc *RabbitMQComponent) SetRequeue(requeue bool) {
	rc.requeue = requeue
}

func (rc *RabbitMQComponent) LoadFromPrefix(prefix string) {
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

func (rc *RabbitMQComponent) Client() *rabbitmq.Client {
	return rc.ensureClient()
}

func (rc *RabbitMQComponent) Publish(ctx context.Context, body []byte, options ...rabbitmq.PublishOption) error {
	return rc.PublishTo(ctx, rc.exchange, rc.routingKey, body, options...)
}

func (rc *RabbitMQComponent) PublishTo(ctx context.Context, exchange, routingKey string, body []byte, options ...rabbitmq.PublishOption) error {
	if routingKey == "" {
		return fmt.Errorf("rabbitmq routing key is required")
	}
	return rc.ensureClient().Publish(ctx, exchange, routingKey, body, options...)
}

func (rc *RabbitMQComponent) Subscribe(handler rabbitmq.Handler) {
	rc.SubscribeQueue(rc.queue, handler)
}

func (rc *RabbitMQComponent) SubscribeQueue(queue string, handler rabbitmq.Handler) {
	if queue == "" {
		panic("rabbitmq queue is required")
	}
	if handler == nil {
		panic("rabbitmq handler is required")
	}

	runner := rabbitmq.NewConsumerRunner(rc.ensureClient(), queue, handler, rabbitmq.ConsumerOptions{
		ConsumerTag: rc.consumerTag,
		AutoAck:     rc.autoAck,
		Prefetch:    rc.prefetch,
		Requeue:     rc.requeue,
	})
	rc.consumerRunners = append(rc.consumerRunners, runner)
}

func (rc *RabbitMQComponent) Start() {
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

func (rc *RabbitMQComponent) Stop() {
	for _, runner := range rc.consumerRunners {
		runner.Stop()
	}
}

func (rc *RabbitMQComponent) AwaitTermination(ctx context.Context) {
	for _, runner := range rc.consumerRunners {
		runner.AwaitTermination(ctx)
	}
}

func (rc *RabbitMQComponent) RunningTasks() int {
	total := 0
	for _, runner := range rc.consumerRunners {
		total += runner.RunningTasks()
	}
	return total
}

func (rc *RabbitMQComponent) Close() error {
	if rc.client == nil {
		return nil
	}
	err := rc.client.Close()
	rc.client = nil
	return err
}

func (rc *RabbitMQComponent) ensureClient() *rabbitmq.Client {
	if rc.client != nil {
		return rc.client
	}
	if rc.url == "" {
		panic("rabbitmq url is required")
	}
	rc.client = rabbitmq.New(rc.url)
	return rc.client
}

func rabbitMQComponentName(name string) string {
	if name == "" {
		return defaultRabbitMQComponentName
	}
	return defaultRabbitMQComponentName + ":" + name
}

func rabbitMQHostPort(rawURL string) string {
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
