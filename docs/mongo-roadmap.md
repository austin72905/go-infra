# Mongo Roadmap

這份文件整理 `go-infra` 目前 Mongo component 的能力邊界，以及下一步建議補齊的功能。

## Current Scope

目前 `go-infra` 已具備的 Mongo 能力：

- `a.Mongo()` / `a.Mongo(name)` component access
- `SetURI(...)`
- `ReadPreference(...)`
- `Timeout(...)`
- `PoolSize(min, max)`
- `MaxConnecting(...)`
- `TLSConfig(...)`
- `SlowOperationThreshold(...)`
- `LoadFromPrefix("mongo")`
- `ForceStart()`
- `Client() *mongo.Client`
- `Ping(ctx)` / `MustPing(ctx)`
- shutdown lifecycle 自動 close

這一版定位是：

- 提供最小可用的 Mongo infra
- 保留 Mongo 比較自然的 `named instance + read preference` 模式
- 不硬做 `master/slave` 抽象
- 暫時不追求和 `std-library` 同等完整度

## Priority 1

這些功能最值得先補，補完後實戰價值會明顯提升：

1. 啟動期初始化策略整理
   - 明確規範哪些服務要 `ForceStart()`
   - 哪些可以只 lazy connect

2. basic migration hook
   - 不一定綁死在 component 內
   - 但可提供標準接點讓服務在 `ForceStart()` 後初始化 migration

3. slow operation instrumentation
   - 目前只有 API 保留，還沒真的接進 driver monitor
   - 這是 Mongo 很值得保留的觀測能力

## Priority 2

這些功能值得保留，但可以等第一波穩定後再補：

1. IAM auth
   - 若有 AWS / Atlas 類場景再補

2. monitor / pool metrics
   - connection pool、slow query、command monitor

3. helper for database selection
   - 如果之後常見 pattern 需要固定 DB name，可再補一層 helper

## Not Urgent

這些能力原本框架有價值，但目前不建議太早搬進 `go-infra`：

1. repository abstraction
2. collection wrapper 大全
3. migration 機制直接綁死在 component 內
4. 強行做 `master/slave` API

原因：

- 容易讓 `MongoComponent` 太快膨脹
- Mongo 比較適合用 `ReadPreference` 表達讀策略
- repository / collection 仍然更適合留給業務層自己決定

## Suggested Target API

建議下一版 Mongo component 至少長成這樣：

- `SetURI(uri string) *MongoComponent`
- `ReadPreference(rp *readpref.ReadPref) *MongoComponent`
- `Timeout(timeout time.Duration) *MongoComponent`
- `PoolSize(min, max uint64) *MongoComponent`
- `MaxConnecting(max uint64) *MongoComponent`
- `TLSConfig(conf *tls.Config) *MongoComponent`
- `LoadFromPrefix(prefix string)`
- `ForceStart()`
- `Client() *mongo.Client`
- `Ping(ctx) error`
- `Close() error`

## Example Properties

```properties
mongo.uri=mongodb://localhost:27017
mongo.timeout=120s
mongo.pool.min=10
mongo.pool.max=50
mongo.max.connecting=50
mongo.read.preference=secondaryPreferred
```

如果之後有 named instance，可往這種方向擴充：

```properties
mongo.analytics.uri=mongodb://localhost:27017
mongo.analytics.read.preference=primary
```
