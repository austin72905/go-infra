# Redis Capability Guidelines

這份文件整理哪些建立在 Redis 之上的能力適合放進 `go-infra`，以及它們的優先順序與判斷原則。

## 核心判斷原則

如果某個 Redis 能力符合以下特徵，通常就適合進 `go-infra`：

- 語意清楚
- 與業務領域無關
- 多個服務都可能重複使用
- 若每個服務各自手寫，容易不一致或出錯
- 值得由 framework 提供一致做法

如果某個能力容易和產品模型、登入狀態、使用者身份或服務專屬協議深度耦合，就不適合太早收進 infra。

## 基礎 Redis 與高階能力的分界

`RedisComponent` 本身負責的是：

- Redis client 建立
- properties 載入
- lifecycle
- probe
- shutdown close

它解的是：

**怎麼安全、穩定地使用 Redis client**

而像 `LockManager`、`RateLimiter` 這些更高階能力，解的是：

**怎麼基於 Redis 實作某種穩定、可重複使用的併發/狀態語義**

所以不需要把所有 `GET/SET/DEL` 再額外封一層；真正值得抽的是有清楚語意的模式。

## 適合進 go-infra 的能力

### 1. LockManager

狀態：已落地

適合程度：非常高

原因：

- 多服務常見
- 語意穩定
- 錯誤實作風險高
- 很適合集中實作 ownership token、TTL、release safety

目前 `go-infra` 已做到：

- `Acquire(...)` / `TryLock(...)`
- `WithLock(...)`
- `Refresh(...)`
- `Release(...)`
- TTL
- ownership token
- context-aware acquire timeout
- retry/backoff option
- optional auto refresh
- same-key process-local FIFO queue
- typed lock errors

使用方式與設計筆記見：

- redis-lock-manager-usage.md
- lock-manager-design-notes.md

### 2. RateLimiter

適合程度：高

原因：

- 限流屬於典型 infra 能力
- 可跨 HTTP / gRPC / background task 共用
- 若各服務自己做，策略和 key 規則容易不一致

建議做到：

- key-based limit
- window/ttl-based limit
- context-aware API

### 3. Deduplicator

狀態：已落地

適合程度：高

原因：

- 常見在事件去重、短時間防重、任務重入保護
- 語意清楚
- 很適合用 Redis TTL key 實作

目前 `go-infra` 已做到：

- `TryMark(...)`
- `Marked(...)`
- `Clear(...)`
- TTL
- named Redis instance 綁定
- typed errors

使用方式與設計筆記見：

- deduplicator-usage.md
- deduplicator-design-notes.md

## 視使用密度決定的能力

### 4. CacheHelper

適合程度：中高

前提：

- 多個服務真的有大量 cache get/set 模式
- 希望 key naming、TTL、serialize/deserialize 一致

適合做的部分：

- key helper
- TTL convention
- serialize / deserialize helper
- optional singleflight

不建議做的部分：

- 把 Redis 全部指令再包一層

### 5. Counter

適合程度：中

前提：

- 專案中確實有很多通用計數場景
- 希望計數 key 規則與 TTL 規則統一

如果只是偶爾 `IncrBy`，通常不值得專門抽象。

## 不建議太早進 go-infra 的能力

### 6. SessionStore

適合程度：低到中

原因：

- 很容易和 auth / login / token / gateway 規則綁死
- 不同服務對 session 的需求差異可能很大
- 太早抽很容易變成 everyone 都要繞過的 abstraction

只有在你們 session 模式已經非常穩定時，才適合收進 infra。

## 優先順序建議

如果未來要做 Redis 高階能力，我建議順序是：

1. `RateLimiter`
2. `CacheHelper`
3. `Counter`
4. `SessionStore`

## 一句話總結

適合放進 `go-infra` 的 Redis 能力，不是單純 `GET/SET` 指令封裝，而是那些：

- 有明確語意
- 和業務無關
- 跨服務重複
- 很值得統一實作

目前已完成且最值得優先落地的是：

- `LockManager`

而下一批較值得考慮的是：

- `RateLimiter`
