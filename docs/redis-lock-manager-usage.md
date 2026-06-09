# RedisLockManager Usage

這份文件整理 `go-infra` 中 `RedisLockManager` 的定位、用法，以及適合使用的場景。

## 定位

`RedisLockManager` 是建立在 `RedisComponent` 之上的高階能力。

它負責的是：

- 分散式互斥
- 安全釋放
- 同 key 的本機競爭收斂
- retry/backoff
- 可選的自動續期

它不負責：

- 通用 Redis 指令封裝
- session 模型
- 業務專屬 key 規範

## 與 RedisComponent 的關係

`RedisLockManager` 預設共用既有 `RedisComponent` 的連線與設定。

例如：

```go
func (a *App) Initialize() {
	a.Redis("cache").LoadFromPrefix("redis.cache")
	a.RedisLock("cache")
}
```

這代表：

- `a.RedisLock("cache")` 會使用 `a.Redis("cache")`
- 不需要再維護一套 `redis.lock.*` 設定

## 基本 API

### `TryLock`

適合：

- cron
- housekeeping
- cluster 中只想有一台執行

特性：

- 單次嘗試
- 搶不到就回 `ok=false`
- 不長時間等待

```go
lock, ok, err := a.RedisLock().TryLock(ctx, "job:cleanup", 30*time.Second)
if err != nil {
	return err
}
if !ok {
	return nil
}
defer func() { _ = lock.Release(ctx) }()
```

### `WithLock`

適合：

- 一般臨界區
- 希望降低漏掉 release 風險的場景

特性：

- acquire
- 執行 callback
- 自動 release

```go
err := a.RedisLock().WithLock(ctx, "player:123:claim", 10*time.Second, func(ctx context.Context) error {
	return doCriticalSection(ctx)
})
```

### `Acquire`

適合：

- 較長流程
- 需要手動控制 release
- 想拿到鎖後再做 double-check state

```go
lock, err := a.RedisLock().Acquire(
	ctx,
	"token:vendor:x",
	15*time.Second,
	module.WithBackoff(100*time.Millisecond),
	module.WithMaxRetries(10),
)
if err != nil {
	return err
}
defer func() { _ = lock.Release(ctx) }()
```

## 推薦使用情境

### 1. cron / housekeeping

用 `TryLock`。

因為這類任務通常只需要：

- 同一時間只有一台跑
- 搶不到就算了

### 2. 一般臨界區

用 `WithLock`。

例如：

- 同一玩家短時間只能進一次流程
- 同一筆單據不能重複執行

### 3. token refresh / expensive recompute

用 `Acquire + double-check state`。

做法：

1. 先 acquire
2. 拿到鎖後再查一次共享狀態
3. 若前人已更新完成，就直接返回
4. 否則才做重操作

這樣可以避免：

- 拿到鎖就重做昂貴工作
- 多台機器串行重複刷新同一份資料

## Option

第一版提供：

- `WithBackoff(...)`
- `WithMaxRetries(...)`
- `WithRetryStrategy(...)`
- `WithAutoRefresh()`

其中：

- `WithAutoRefresh()` 會在持鎖期間背景續期
- 適合執行時間可能超過 TTL 的場景

## 錯誤語意

需要特別注意的錯誤：

- `ErrLockNotAcquired`
- `ErrLockNotHeld`
- `ErrLockReleased`

若要分類處理，可搭配 `errors.Is(...)`。

## 使用原則

- 需要 lifecycle 管理的 named lock manager，請在 `Initialize()` 期間先註冊
- 不要把 `RedisLockManager` 當成通用 Redis helper
- lock key 請保持可讀、穩定、可觀測
- 若拿到鎖後還能再查共享狀態，優先做 double-check
