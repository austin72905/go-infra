package kafka

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

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

type Producer struct {
	inner      *ckafka.Producer
	onDelivery OnDelivery
	wg         sync.WaitGroup
}

func NewProducer(brokers []string) *Producer {
	return NewProducerWithDelivery(brokers, nil)
}

func NewProducerWithDelivery(brokers []string, onDelivery OnDelivery) *Producer {
	if len(brokers) == 0 {
		panic("kafka brokers are required")
	}

	producer, err := ckafka.NewProducer(&ckafka.ConfigMap{
		"bootstrap.servers": strings.Join(brokers, ","),
	})
	if err != nil {
		panic(fmt.Sprintf("create kafka producer failed: %v", err))
	}

	p := &Producer{
		inner:      producer,
		onDelivery: onDelivery,
	}
	p.startDeliveryLoop()
	return p
}

func (p *Producer) Publish(ctx context.Context, topic string, key, value []byte) error {
	err := p.inner.Produce(&ckafka.Message{
		TopicPartition: ckafka.TopicPartition{
			Topic:     &topic,
			Partition: ckafka.PartitionAny,
		},
		Key:   key,
		Value: value,
	}, nil)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (p *Producer) SyncPublish(ctx context.Context, topic string, key, value []byte) error {
	deliveryCh := make(chan ckafka.Event, 1)
	defer close(deliveryCh)

	err := p.inner.Produce(&ckafka.Message{
		TopicPartition: ckafka.TopicPartition{
			Topic:     &topic,
			Partition: ckafka.PartitionAny,
		},
		Key:   key,
		Value: value,
	}, deliveryCh)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case event := <-deliveryCh:
		message, ok := event.(*ckafka.Message)
		if !ok {
			return fmt.Errorf("unexpected kafka delivery event %T", event)
		}
		return message.TopicPartition.Error
	}
}

func (p *Producer) Close() {
	if p.inner == nil {
		return
	}
	p.inner.Flush(5000)
	p.inner.Close()
	p.wg.Wait()
}

func (p *Producer) startDeliveryLoop() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		for event := range p.inner.Events() {
			message, ok := event.(*ckafka.Message)
			if !ok || p.onDelivery == nil {
				continue
			}

			p.onDelivery(DeliveryReport{
				Topic:     topicName(message.TopicPartition.Topic),
				Key:       append([]byte(nil), message.Key...),
				Value:     append([]byte(nil), message.Value...),
				Partition: message.TopicPartition.Partition,
				Offset:    int64(message.TopicPartition.Offset),
			}, message.TopicPartition.Error)
		}
	}()
}

type ConsumerRunner struct {
	consumer *ckafka.Consumer
	handler  Handler
	topic    string
	offset   int64
	queue    chan *ckafka.Message

	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	runningTasks atomic.Int32
}

func NewConsumerRunner(brokers []string, groupID, topic string, handler Handler) *ConsumerRunner {
	return NewConsumerRunnerWithOffsetAndQueue(brokers, groupID, topic, OffsetEarliest, 0, handler)
}

func NewConsumerRunnerWithOffset(brokers []string, groupID, topic string, offset int64, handler Handler) *ConsumerRunner {
	return NewConsumerRunnerWithOffsetAndQueue(brokers, groupID, topic, offset, 0, handler)
}

func NewConsumerRunnerWithOffsetAndQueue(brokers []string, groupID, topic string, offset int64, queueCapacity int, handler Handler) *ConsumerRunner {
	if len(brokers) == 0 {
		panic("kafka brokers are required")
	}
	if groupID == "" {
		panic("kafka group id is required")
	}
	if topic == "" {
		panic("kafka topic is required")
	}
	if handler == nil {
		panic("kafka handler is required")
	}

	consumer, err := ckafka.NewConsumer(&ckafka.ConfigMap{
		"bootstrap.servers":  strings.Join(brokers, ","),
		"group.id":           groupID,
		"auto.offset.reset":  offsetResetPolicy(offset),
		"enable.auto.commit": false,
	})
	if err != nil {
		panic(fmt.Sprintf("create kafka consumer failed: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &ConsumerRunner{
		consumer: consumer,
		handler:  handler,
		topic:    topic,
		offset:   offset,
		queue:    make(chan *ckafka.Message, max(queueCapacity, 0)),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (r *ConsumerRunner) Start() {
	if err := r.subscribe(); err != nil {
		panic(fmt.Sprintf("subscribe kafka topic %q failed: %v", r.topic, err))
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer close(r.queue)

		for {
			select {
			case <-r.ctx.Done():
				return
			default:
			}

			event := r.consumer.Poll(250)
			if event == nil {
				continue
			}

			switch message := event.(type) {
			case *ckafka.Message:
				select {
				case <-r.ctx.Done():
					return
				case r.queue <- message:
				}
			case ckafka.Error:
				continue
			default:
				continue
			}
		}
	}()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		for {
			select {
			case <-r.ctx.Done():
				return
			case message, ok := <-r.queue:
				if !ok {
					return
				}

				r.runningTasks.Add(1)
				err := r.handler(r.ctx, Message{
					Topic:     topicName(message.TopicPartition.Topic),
					Key:       append([]byte(nil), message.Key...),
					Value:     append([]byte(nil), message.Value...),
					Partition: message.TopicPartition.Partition,
					Offset:    int64(message.TopicPartition.Offset),
				})
				r.runningTasks.Add(-1)
				if err == nil {
					_, _ = r.consumer.CommitMessage(message)
				}
			}
		}
	}()
}

func (r *ConsumerRunner) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	if r.consumer != nil {
		_ = r.consumer.Close()
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

func (r *ConsumerRunner) subscribe() error {
	if r.offset == OffsetLatest || r.offset == OffsetEarliest {
		return r.consumer.SubscribeTopics([]string{r.topic}, nil)
	}

	metadata, err := r.consumer.GetMetadata(&r.topic, false, 5000)
	if err != nil {
		return err
	}

	topicMetadata, ok := metadata.Topics[r.topic]
	if !ok {
		return fmt.Errorf("topic metadata not found for %q", r.topic)
	}

	partitions := make([]ckafka.TopicPartition, 0, len(topicMetadata.Partitions))
	for _, partition := range topicMetadata.Partitions {
		partitions = append(partitions, ckafka.TopicPartition{
			Topic:     &r.topic,
			Partition: partition.ID,
			Offset:    ckafka.Offset(r.offset),
		})
	}

	return r.consumer.Assign(partitions)
}

func offsetResetPolicy(offset int64) string {
	if offset == OffsetLatest {
		return "latest"
	}
	return "earliest"
}

func topicName(topic *string) string {
	if topic == nil {
		return ""
	}
	return *topic
}

func max(value, fallback int) int {
	if value < fallback {
		return fallback
	}
	return value
}
