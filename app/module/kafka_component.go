package module

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	internalkafka "go-infra/internal/kafka"
	internalmodule "go-infra/internal/module"
)

const defaultKafkaComponentName = "Kafka"

type KafkaComponent struct {
	appRuntime *AppRuntime

	brokers       []string
	groupID       string
	poolSize      int
	queueCapacity int

	onDelivery         internalkafka.OnDelivery
	producer           *internalkafka.Producer
	consumerRunners    []*internalkafka.ConsumerRunner
	startRegistered    bool
	stopRegistered     bool
	closeProducerAdded bool
}

func (kc *KafkaComponent) initialize(runtime *AppRuntime, name string) {
	kc.appRuntime = runtime

	if !kc.startRegistered {
		runtime.Lifecycle.Startup.AddStart(internalmodule.TaskFunc(func(ctx context.Context) {
			kc.Start()
		}))
		kc.startRegistered = true
	}

	if !kc.stopRegistered {
		runtime.Lifecycle.Shutdown.AddStopBackground(internalmodule.TaskFunc(func(ctx context.Context) {
			kc.Stop()
		}))
		runtime.Lifecycle.Shutdown.AddAwaitBackground(internalmodule.TaskFunc(func(ctx context.Context) {
			kc.AwaitTermination(ctx)
		}))
		kc.stopRegistered = true
	}

	if !kc.closeProducerAdded {
		runtime.Lifecycle.Shutdown.AddCloseProducers(internalmodule.TaskFunc(func(ctx context.Context) {
			kc.CloseProducer()
		}))
		kc.closeProducerAdded = true
	}
}

func (kc *KafkaComponent) validate() {
}

func (kc *KafkaComponent) SetBrokers(brokers ...string) {
	kc.brokers = kc.brokers[:0]
	for _, broker := range brokers {
		trimmed := strings.TrimSpace(broker)
		if trimmed == "" {
			continue
		}
		kc.brokers = append(kc.brokers, trimmed)
		kc.appRuntime.Probe.AddHostURI(trimmed)
	}
}

func (kc *KafkaComponent) SetGroupID(groupID string) {
	kc.groupID = groupID
}

func (kc *KafkaComponent) SetPoolSize(poolSize int) {
	if poolSize <= 0 {
		poolSize = 1
	}
	kc.poolSize = poolSize
}

func (kc *KafkaComponent) SetQueueCapacity(queueCapacity int) {
	if queueCapacity < 0 {
		queueCapacity = 0
	}
	kc.queueCapacity = queueCapacity
}

func (kc *KafkaComponent) LoadFromPrefix(prefix string) {
	if prefix == "" {
		panic("kafka property prefix is required")
	}

	brokers := kc.appRuntime.Property.RequiredProperty(prefix + ".brokers")
	kc.SetBrokers(strings.Split(brokers, ",")...)
	kc.SetGroupID(kc.appRuntime.Property.Property(prefix + ".group.id"))

	poolSizeValue := kc.appRuntime.Property.Property(prefix + ".pool.size")
	if poolSizeValue == "" {
		kc.SetPoolSize(1)
		return
	}

	poolSize, err := strconv.Atoi(poolSizeValue)
	if err != nil {
		panic(fmt.Sprintf("invalid kafka pool size for prefix %q: %v", prefix, err))
	}
	kc.SetPoolSize(poolSize)

	queueCapacityValue := kc.appRuntime.Property.Property(prefix + ".queue.capacity")
	if queueCapacityValue == "" {
		return
	}

	queueCapacity, err := strconv.Atoi(queueCapacityValue)
	if err != nil {
		panic(fmt.Sprintf("invalid kafka queue capacity for prefix %q: %v", prefix, err))
	}
	kc.SetQueueCapacity(queueCapacity)
}

func (kc *KafkaComponent) Publish(ctx context.Context, topic string, value []byte, key ...[]byte) error {
	if topic == "" {
		return fmt.Errorf("kafka topic is required")
	}

	var messageKey []byte
	if len(key) > 0 {
		messageKey = key[0]
	}

	return kc.ensureProducer().Publish(ctx, topic, messageKey, value)
}

func (kc *KafkaComponent) SyncPublish(ctx context.Context, topic string, value []byte, key ...[]byte) error {
	if topic == "" {
		return fmt.Errorf("kafka topic is required")
	}

	var messageKey []byte
	if len(key) > 0 {
		messageKey = key[0]
	}

	return kc.ensureProducer().SyncPublish(ctx, topic, messageKey, value)
}

func (kc *KafkaComponent) OnDelivery(onDelivery internalkafka.OnDelivery) {
	kc.onDelivery = onDelivery
}

func (kc *KafkaComponent) Subscribe(topic string, handler internalkafka.Handler) {
	kc.SubscribeByOffset(topic, internalkafka.OffsetEarliest, handler)
}

func (kc *KafkaComponent) SubscribeByOffset(topic string, offset int64, handler internalkafka.Handler) {
	if handler == nil {
		panic("kafka handler is required")
	}
	if topic == "" {
		panic("kafka topic is required")
	}
	if len(kc.brokers) == 0 {
		panic("kafka brokers are required")
	}
	if kc.groupID == "" {
		panic("kafka group id is required")
	}

	poolSize := kc.poolSize
	if poolSize <= 0 {
		poolSize = 1
	}

	for range poolSize {
		runner := internalkafka.NewConsumerRunnerWithOffsetAndQueue(kc.brokers, kc.groupID, topic, offset, kc.queueCapacity, handler)
		kc.consumerRunners = append(kc.consumerRunners, runner)
	}
}

func (kc *KafkaComponent) Start() {
	for _, runner := range kc.consumerRunners {
		runner.Start()
	}
}

func (kc *KafkaComponent) Stop() {
	for _, runner := range kc.consumerRunners {
		runner.Stop()
	}
}

func (kc *KafkaComponent) AwaitTermination(ctx context.Context) {
	for _, runner := range kc.consumerRunners {
		runner.AwaitTermination(ctx)
	}
}

func (kc *KafkaComponent) RunningTasks() int {
	total := 0
	for _, runner := range kc.consumerRunners {
		total += runner.RunningTasks()
	}
	return total
}

func (kc *KafkaComponent) CloseProducer() {
	if kc.producer == nil {
		return
	}
	kc.producer.Close()
	kc.producer = nil
}

func (kc *KafkaComponent) ensureProducer() *internalkafka.Producer {
	if kc.producer != nil {
		return kc.producer
	}
	if len(kc.brokers) == 0 {
		panic("kafka brokers are required")
	}

	kc.producer = internalkafka.NewProducerWithDelivery(kc.brokers, kc.onDelivery)
	return kc.producer
}

func kafkaComponentName(name string) string {
	if name == "" {
		return defaultKafkaComponentName
	}
	return defaultKafkaComponentName + ":" + name
}
