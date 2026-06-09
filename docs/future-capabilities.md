# Future Capabilities

這份文件整理 `go-infra` 未來**適合考慮加入 infra 層**的能力，目的是保留擴充方向，但避免現在就把框架做得太厚。

## 判斷原則

適合放進 `infra` 的能力，通常符合這幾個條件：

- 跨服務重複出現
- 和業務領域無關
- 啟動 / 關閉生命週期穩定
- 配置模式穩定
- 值得由 framework 統一規範

不適合太早放進 `infra` 的能力，通常有這些特徵：

- 高度依賴業務模型
- 每個服務差異很大
- 很難抽成一致 API
- 一旦抽錯，後續服務都得繞過 framework

## Priority 1

這些是最值得優先考慮的。

### 1. Logging

建議收進 infra 的部分：

- logger 初始化
- `ctx` 內共用 metadata 的 log helper
- request ID / correlation ID / trace 的統一 log 欄位
- shutdown 前 flush

先不要太早做的：

- 業務層 log 包裝 DSL
- 強綁特定 log 格式到每個 service function
- 把整顆 action log runtime 直接塞進所有 context

原因：

- 幾乎所有服務都需要
- 已經和 `RequestContext`、`Recovery`、`Kafka`、`gRPC` 有天然關聯
- 做得好會直接提升除錯效率

### 2. Metrics / Observability

建議收進 infra 的部分：

- Prometheus registry 初始化
- HTTP / gRPC / Kafka / DB / Redis 的基礎 metrics
- startup / health / readiness 指標
- panic / error counter

先不要太早做的：

- 每個 component 都包很重的 metrics DSL
- 業務報表型 metrics

原因：

- 和 infra component 關聯度高
- 能快速補齊可觀測性

### 3. gRPC Client

這塊目前已經有第一版 `GrpcClientComponent`，後續可繼續擴充。

下一步適合補的：

- `LoadFromPrefix(...)`
- 更清楚的 typed client helper 文件
- retry / timeout / keepalive 預設策略整理
- metrics / tracing interceptor

先不要太早做的：

- 全域 service discovery 抽象
- 過度複雜的 load balancing 策略

### 4. HTTP Client

建議收進 infra 的部分：

- 共用 `http.Client` 建立
- timeout / keepalive / transport pool 設定
- request ID / correlation ID 傳遞
- retry / backoff 基礎能力
- shutdown lifecycle

先不要太早做的：

- 包一整套 REST SDK framework
- 強制每個下游都走同一種回應抽象

原因：

- 和 gRPC client 類似，屬於很常重複的基礎能力
- 對單體與微服務都適用

## Priority 2

這些值得做，但不需要比前面更早。

### 5. Tracing

建議收進 infra 的部分：

- OpenTelemetry 初始化
- exporter 設定
- HTTP / gRPC client/server propagation
- trace 與 `RequestContext` 結合

先不要太早做的：

- 一開始就做太多 vendor-specific 綁定

原因：

- 很有價值
- 但最好等 logging / metrics 的基本盤穩了再做

### 6. Background Worker / Executor

建議收進 infra 的部分：

- worker pool
- background task lifecycle
- goroutine 啟停規範
- graceful shutdown 等待

先不要太早做的：

- 把所有非同步業務都塞進同一個 executor abstraction

原因：

- 你目前已經有 `Scheduler`、`Kafka`，之後很可能會需要共用背景工作模式

### 7. Cache Helper

建議收進 infra 的部分：

- cache key 命名 helper
- serialize / deserialize helper
- TTL convention
- singleflight 保護

先不要太早做的：

- Redis 全 API 再包一層

原因：

- 很多服務會把 Redis 當 cache
- 但不值得把 Redis 原生能力全部抽象化

### 8. Redis Lock Manager

建議收進 infra 的部分：

- `Acquire(...)` / `TryLock(...)`
- `Release(...)`
- lock ownership token
- TTL
- context-aware acquire timeout

第二波可考慮的：

- `Refresh(...)`
- 可選的 lock key helper

先不要太早做的：

- leader election
- 很厚的 lock orchestration
- 自動背景 renew 全家桶
- 和業務 key 命名強綁的設計

原因：

- lock manager 屬於高價值、跨服務重複的能力
- 行為模式相對固定
- 寫錯風險高，集中在 infra 反而比較安全
- 但目前不需要比 logging / metrics / client 類能力更早

### 9. Feature Flags

建議收進 infra 的部分：

- config-based feature flag
- typed flag helper
- runtime reload（若未來有這需求）

先不要太早做的：

- 一開始就做成完整旗標平台

## Priority 3

這些可以記著，但不建議太早做。

### 10. Middleware / Interceptor Helpers

適合保留在 infra 的：

- HTTP recovery
- request metadata middleware
- gRPC recovery interceptor
- gRPC metadata interceptor
- shutdown / max-connections interceptor

不建議太早做厚的：

- 一大包把 auth / tracing / metrics / validation 全綁死的 middleware suite

原因：

- 這類能力有價值
- 但最容易一路長成 framework 大泥球

### 11. Typed Config Helpers

建議收進 infra 的部分：

- `Int(...)`
- `Bool(...)`
- `Duration(...)`
- `StringSlice(...)`
- 帶 default 值的 helper

原因：

- `PropertyRuntime` 已經有基礎
- 補 typed helper 成本低、收益高

## 暫時不建議放進 infra

以下能力目前不建議太早收進 `go-infra`：

- websocket 完整框架
- repository abstraction
- ORM query DSL
- 業務 service base class
- domain event model
- 與特定產品業務強綁的 auth/user model

原因：

- 這些高度依賴服務本身
- 抽太早很容易抽錯

## 建議實作順序

如果要在現有基礎上繼續擴充，我會建議順序是：

1. Logging
2. Metrics / Observability
3. 補強 gRPC Client
4. HTTP Client
5. Tracing
6. Background Worker

## 總結

`go-infra` 應該優先承接的是：

- 所有服務都會重複碰到
- 但不該每個服務自己手刻
- 且生命週期與配置模式都相對穩定

這樣框架會越來越有價值，而不是越來越厚。
