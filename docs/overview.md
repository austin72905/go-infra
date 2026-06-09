# go-infra Overview

這份文件整理 `go-infra` 目前的整體架構、已完成能力、各 component 的責任邊界，以及下一步建議。

## Design Goal

`go-infra` 的目標不是複製一個超厚的 `std-library`，而是先做出一套：

- 命名清楚
- lifecycle 明確
- properties 載入一致
- 可逐步擴充
- 不過早抽象

的微服務基礎設施骨架。

目前的設計核心是：

- `AppRuntime`
- `LifecycleManager`
- `ModuleBase`
- `ComponentRegistry`
- 各種 infra `Component`

## Current Architecture

### AppRuntime

`AppRuntime` 是整個 app 的執行期容器，負責收納 framework 核心能力：

- `Lifecycle`
- `Components`
- `Web`
- `Property`
- `Probe`

它是 runtime 本體，不是單一基礎設施元件。

### Lifecycle

`go-infra` 目前有兩條主要 lifecycle：

- `StartupLifecycle`
  - `Initialize`
  - `Start`
  - `Serve`
- `ShutdownLifecycle`
  - `StopIngress`
  - `AwaitRequests`
  - `StopBackground`
  - `AwaitBackground`
  - `CloseProducers`
  - `ReleaseResources`
  - `StopServer`

framework 會在 `App.Start()` 中統一執行這兩條流程。

### ModuleBase

`ModuleBase` 是服務端會直接使用的 facade。

服務通常透過它取得 infra component，例如：

- `a.DB()`
- `a.Redis()`
- `a.Kafka()`
- `a.Mongo()`
- `a.Schedule()`
- `a.Grpc()`

### Component Registry

`ComponentRegistry` 只負責：

- register component
- lookup component

它刻意保持薄，不做 typed service locator 大總管。

## Current Components

### Runtime-level

這些比較接近 app runtime 本體：

- `PropertyRuntime`
- `StartupProbe`
- `WebRuntime`

### Optional Infra Components

這些屬於可選能力，需要在 `Initialize()` 期間先註冊：

- `SchedulerComponent`
- `RedisComponent`
- `RedisLockManager`
- `Deduplicator`
- `DBComponent`
- `KafkaComponent`
- `GrpcComponent`
- `MongoComponent`

這些 component 若在 app startup 後才第一次建立，會直接 panic，目的是避免錯過 lifecycle 註冊。

## Component Status

### WebRuntime

狀態：可用初版

已具備：

- Gin engine
- `Listen(...)`
- `Router()`
- startup 自動啟動
- shutdown 自動關閉

目前策略：

- 只支援 Gin
- 不過早做厚的 server abstraction

### SchedulerComponent

狀態：可用初版

已具備：

- function job
- named function job
- lifecycle start / stop / await termination
- running task tracking

目前策略：

- 以 function API 為主
- 不急著回頭做厚的 job abstraction

### RedisComponent

狀態：可用初版

已具備：

- `Redis(name...)`
- `LoadFromPrefix(...)`
- `Client()`
- `Ping()`
- `Close()`
- shutdown close
- startup probe host registration

目前策略：

- framework 只做 client management
- Redis command API 仍交給底層 client

### RedisLockManager

狀態：第一版實戰可用

已具備：

- `RedisLock(name...)`
- `TryLock(...)`
- `Acquire(...)`
- `WithLock(...)`
- owner token
- Lua 安全釋放 / 續期
- retry / backoff option
- optional auto refresh
- process-local same-key FIFO queue
- typed lock errors

目前策略：

- `RedisLockManager` 是高階能力，綁既有 `RedisComponent`
- 不新增獨立 `redis.lock.*` properties
- 先不做 Redlock，多節點鎖留待後續

### Deduplicator

狀態：第一版實戰可用

已具備：

- `Deduplicator(name...)`
- `TryMark(...)`
- `Marked(...)`
- `Clear(...)`
- TTL-based short-term dedupe
- named Redis instance 綁定
- typed dedupe errors

目前策略：

- `Deduplicator` 解短期防重，不承諾永久唯一性
- 不新增獨立 `redis.dedupe.*` properties
- Redis 失敗直接回 error，由上層決定 fallback

### DBComponent

狀態：可用初版

已具備：

- PostgreSQL 導向實作
- `LoadFromPrefix(...)`
- `SQLDB()`
- `GormDB()`
- pool config
- `Ping()`
- `Close()`
- named instance

目前策略：

- framework 管連線與 lifecycle
- query / transaction / repository 邏輯交給呼叫端

### KafkaComponent

狀態：第一版實戰可用

已具備：

- `Publish(...)`
- `SyncPublish(...)`
- `OnDelivery(...)`
- `Subscribe(...)`
- `SubscribeByOffset(...)`
- `SetPoolSize(...)`
- `SetQueueCapacity(...)`
- `LoadFromPrefix(...)`
- lifecycle start / stop / await / close producer
- running task tracking

詳細規劃見：

- kafka-roadmap.md

### GrpcComponent

狀態：可用初版

已具備：

- `Listen(...)`
- `LoadFromPrefix(...)`
- `AddOpt(...)`
- `MaxConnections(...)`
- `Server()`
- `HealthServer()`
- lifecycle start / graceful shutdown

詳細規劃見：

- grpc-roadmap.md

### MongoComponent

狀態：可用初版

已具備：

- `Mongo(name...)`
- `SetURI(...)`
- `ReadPreference(...)`
- `Timeout(...)`
- `PoolSize(...)`
- `MaxConnecting(...)`
- `TLSConfig(...)`
- `SlowOperationThreshold(...)`
- `LoadFromPrefix(...)`
- `ForceStart()`
- `Client()`
- `Ping()`
- `Close()`

詳細規劃見：

- mongo-roadmap.md

## Extension Planning Docs

延伸規劃文件目前有：

- future-capabilities.md
- kafka-roadmap.md
- grpc-roadmap.md
- mongo-roadmap.md
- redis-capability-guidelines.md
- lock-manager-design-notes.md
- redis-lock-manager-usage.md
- deduplicator-design-notes.md
- deduplicator-usage.md

## What Belongs in go-infra

比較適合進 `go-infra` 的能力：

- DB
- Redis
- Mongo
- Kafka
- gRPC server
- scheduler
- HTTP server runtime
- properties
- lifecycle
- probe

這些能力的共同點是：

- 啟停規則穩定
- 配置模式穩定
- 各服務間共用度高

## What Should Stay Out for Now

目前不建議太早進 `go-infra` 的能力：

- websocket framework
- repository abstraction
- query helper 大全
- 太厚的 business-aware wrappers
- metrics / logging / tracing 全家桶一次做滿

原因：

- 容易過早抽象
- 容易和業務協議綁太深
- 容易讓 `go-infra` 重新膨脹成重型 framework

## Recommended Usage Pattern

服務端在 `Initialize()` 內先註冊需要的 component：

```go
func (a *App) Initialize() {
	_ = a.LoadProperties(envProperties, "dev", "app.properties")

	a.DB().LoadFromPrefix("db")
	a.Redis().LoadFromPrefix("redis")
	a.Kafka().LoadFromPrefix("kafka")
	a.Mongo().LoadFromPrefix("mongo")
	a.Grpc().LoadFromPrefix("grpc")
	a.Schedule()
}
```

這樣 startup/shutdown lifecycle 才能完整接手。

## Next Suggested Work

如果要繼續擴充，我建議順序是：

1. 補一到兩個實際服務的 sample usage
2. 補基礎測試
3. 規範 properties key 命名
4. 最後再考慮 metrics / logging / tracing

比起再一直新增 component，現在更值得把已完成的能力穩住。
