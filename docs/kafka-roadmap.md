# Kafka Roadmap

這份文件整理 `go-infra` 目前 Kafka component 的能力邊界，以及下一步建議補齊的功能。

## Current Scope

目前 `go-infra` 已具備的 Kafka 能力：

- `a.Kafka()` / `a.Kafka(name)` component access
- `SetBrokers(...)`
- `SetGroupID(...)`
- `SetPoolSize(...)`
- `SetQueueCapacity(...)`
- `LoadFromPrefix("kafka")`
- `Publish(ctx, topic, value, key...)`
- `SyncPublish(ctx, topic, value, key...)`
- `OnDelivery(...)`
- `Subscribe(topic, handler)`
- `SubscribeByOffset(topic, offset, handler)`
- startup lifecycle 自動啟動 consumer
- shutdown lifecycle：
  - stop consumer
  - await consumer termination
  - close producer

這一版定位是：

- 提供最小可用的 Kafka infra
- 先把 component / lifecycle / properties 模式收斂
- 暫時不追求和 `std-library` 同等完整度

## Priority 1

這些功能最值得先補，補完後實戰價值會明顯提升：

1. consumer graceful shutdown 強化
   - 可再補更細的 progress / metrics

## Priority 2

這些功能值得保留，但可以等第一波穩定後再補：

1. `RunningTasks` / progress metrics
   - 提供更好的運行觀測能力

2. named Kafka instances 的使用範例與規範
   - 例如 `a.Kafka("sync")`
   - 適合多 broker cluster 或多用途 Kafka 配置

## Not Urgent

這些能力原本框架有價值，但目前不建議太早搬進 `go-infra`：

1. generic typed handler
2. bulk handler
3. sys management routes
4. payload type registry
5. 過多 producer option 開關

原因：

- 容易讓 `KafkaComponent` 再次膨脹
- 會把 `go-infra` 拉回重型 framework 方向
- 現階段更重要的是先把最小核心能力做穩

## Suggested Target API

建議下一版 Kafka component 至少長成這樣：

- `SetBrokers(...)`
- `SetGroupID(...)`
- `SetPoolSize(...)`
- `SetQueueCapacity(...)`
- `LoadFromPrefix(prefix string)`
- `Publish(ctx, topic, value, key...) error`
- `SyncPublish(ctx, topic, value, key...) error`
- `OnDelivery(onDelivery OnDelivery)`
- `Subscribe(topic string, handler Handler)`
- `SubscribeByOffset(topic string, offset int64, handler Handler)`
- `Stop()`
- `AwaitTermination(ctx)`
- `CloseProducer()`

## Example Properties

```properties
kafka.brokers=localhost:9092,localhost:9093
kafka.group.id=my-group
kafka.pool.size=1
kafka.queue.capacity=1000
```
