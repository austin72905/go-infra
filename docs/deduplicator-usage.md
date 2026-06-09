# Deduplicator Usage

這份文件整理 `go-infra` 中 `Deduplicator` 的定位、用法，以及適合使用的場景。

## 定位

`Deduplicator` 是建立在 `RedisComponent` 之上的高階能力。

它負責的是：

- 短期防重
- TTL 視窗內避免重複處理
- callback / webhook / API / cron 的輕量去重

它不負責：

- 永久唯一性保證
- DB idempotency
- 事件流高吞吐 Bloom 快篩

## 與 RedisComponent 的關係

`Deduplicator` 預設共用既有 `RedisComponent` 的連線與設定。

例如：

```go
func (a *App) Initialize() {
	a.Redis("callback").LoadFromPrefix("redis.callback")
	a.Deduplicator("callback")
}
```

這代表：

- `a.Deduplicator("callback")` 會使用 `a.Redis("callback")`
- 不需要再維護一套 `redis.dedupe.*` 設定

## 基本 API

### `TryMark`

語意：

- 第一次標記成功時回 `true`
- 若 key 在 TTL 內已存在則回 `false`
- Redis 失敗時直接回 error

```go
ok, err := a.Deduplicator().TryMark(ctx, "callback:order:123", 5*time.Minute)
if err != nil {
	return err
}
if !ok {
	return nil
}
```

適合：

- webhook callback 防重
- 通知不要重送
- API 短時間重複提交
- cron 任務短時間不要重跑

### `Marked`

語意：

- 只查目前 key 是否存在
- 不延長 TTL
- 不修改狀態

```go
marked, err := a.Deduplicator().Marked(ctx, "callback:order:123")
if err != nil {
	return err
}
if marked {
	return nil
}
```

適合：

- 想先看某件事最近是否已被處理
- 管理介面或補償流程查當前防重狀態

### `Clear`

語意：

- 主動刪除防重 key

```go
if err := a.Deduplicator().Clear(ctx, "callback:order:123"); err != nil {
	return err
}
```

適合：

- callback rollback
- 補償流程重置
- 測試場景手動清掉防重狀態

## 推薦使用情境

### 1. webhook / callback

用 `TryMark`。

例如：

- 第三方支付 callback
- 外部平台回調
- 重試時可能同一筆通知反覆送來

### 2. API 短時間防重

用 `TryMark`。

例如：

- 前端連點提交
- 同一玩家短時間重覆觸發同一操作

### 3. cron / housekeeping

若只是避免短時間重複執行，可用 `TryMark`。

若需要嚴格互斥，請改用 `RedisLockManager`。

## 和 RedisLockManager 的差別

`Deduplicator` 解的是：

- 這件事最近是否處理過

`RedisLockManager` 解的是：

- 同一時間是否只能有一個執行者進入臨界區

所以：

- dedupe = 短期防重
- lock = 互斥控制

## 和 DB unique key / idempotency table 的差別

`Deduplicator` 是：

- 快
- 輕量
- 短期 TTL 視窗防重

但它不是：

- 永久唯一性保證

如果需求是：

- 這筆 transaction 永遠只能處理一次
- 這筆 order 永遠不能重覆入帳

那還是要靠：

- DB unique key
- idempotency table
- persistent processed-event record

## 使用原則

- named `Deduplicator` 請在 `Initialize()` 期間先註冊
- key 要穩定、可讀、能對應到重複語意
- TTL 由呼叫端根據場景決定
- Redis 失敗時 `TryMark` 直接回 error，由上層決定 fallback
