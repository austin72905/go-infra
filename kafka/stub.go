//go:build !cgo

package kafka

import (
	"context"
	"errors"
)

var ErrCGODisabled = errors.New("kafka requires cgo and a C compiler toolchain")

type Message struct {
	Topic     string
	Key       []byte
	Value     []byte
	Partition int32
	Offset    int64
}

type Handler func(ctx context.Context, message Message) error

type DeliveryReport struct {
	Topic     string
	Key       []byte
	Value     []byte
	Partition int32
	Offset    int64
}

type OnDelivery func(report DeliveryReport, err error)

const (
	OffsetLatest   int64 = -1
	OffsetEarliest int64 = -2
)

type Producer struct{}

type ConsumerRunner struct{}

func NewProducer(brokers []string) *Producer { panic(ErrCGODisabled) }
func NewProducerWithDelivery(brokers []string, onDelivery OnDelivery) *Producer { panic(ErrCGODisabled) }
func (p *Producer) Publish(ctx context.Context, topic string, key, value []byte) error { return ErrCGODisabled }
func (p *Producer) SyncPublish(ctx context.Context, topic string, key, value []byte) error { return ErrCGODisabled }
func (p *Producer) Close() {}

func NewConsumerRunner(brokers []string, groupID, topic string, handler Handler) *ConsumerRunner {
	panic(ErrCGODisabled)
}
func NewConsumerRunnerWithOffset(brokers []string, groupID, topic string, offset int64, handler Handler) *ConsumerRunner {
	panic(ErrCGODisabled)
}
func NewConsumerRunnerWithOffsetAndQueue(brokers []string, groupID, topic string, offset int64, queueCapacity int, handler Handler) *ConsumerRunner {
	panic(ErrCGODisabled)
}
func (r *ConsumerRunner) Start() { panic(ErrCGODisabled) }
func (r *ConsumerRunner) Stop() {}
func (r *ConsumerRunner) AwaitTermination(ctx context.Context) {}
func (r *ConsumerRunner) RunningTasks() int { return 0 }

const DefaultComponentName = "Kafka"

type Component struct {
	appRuntime      any
	brokers         []string
	groupID         string
	poolSize        int
	queueCapacity   int
	onDelivery      OnDelivery
	producer        *Producer
	consumerRunners []*ConsumerRunner
}

func (kc *Component) Initialize(runtime any, name string) {}
func (kc *Component) Validate() {}
func (kc *Component) SetBrokers(brokers ...string) { kc.brokers = append(kc.brokers[:0], brokers...) }
func (kc *Component) SetGroupID(groupID string) { kc.groupID = groupID }
func (kc *Component) SetPoolSize(poolSize int) { kc.poolSize = poolSize }
func (kc *Component) SetQueueCapacity(queueCapacity int) { kc.queueCapacity = queueCapacity }
func (kc *Component) LoadFromPrefix(prefix string) { panic(ErrCGODisabled) }
func (kc *Component) Publish(ctx context.Context, topic string, value []byte, key ...[]byte) error { return ErrCGODisabled }
func (kc *Component) SyncPublish(ctx context.Context, topic string, value []byte, key ...[]byte) error { return ErrCGODisabled }
func (kc *Component) OnDelivery(onDelivery OnDelivery) { kc.onDelivery = onDelivery }
func (kc *Component) Subscribe(topic string, handler Handler) { panic(ErrCGODisabled) }
func (kc *Component) SubscribeByOffset(topic string, offset int64, handler Handler) { panic(ErrCGODisabled) }
func (kc *Component) Start() { panic(ErrCGODisabled) }
func (kc *Component) Stop() {}
func (kc *Component) AwaitTermination(ctx context.Context) {}
func (kc *Component) RunningTasks() int { return 0 }
func (kc *Component) CloseProducer() {}
func ComponentName(name string) string {
	if name == "" {
		return DefaultComponentName
	}
	return DefaultComponentName + ":" + name
}
