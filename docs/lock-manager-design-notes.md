# Lock Manager Design Notes

這份文件整理 `go-infra` 的 Redis lock manager 設計原則。

## 結論

`RedisLockManager` 不只是 `SETNX` 的薄包裝，而是提供一套適合服務共用的分散式鎖語意：

- owner token
- Lua 安全釋放
- Lua 安全續期
- `TryLock` / `Acquire` / `WithLock` 分層
- retry / backoff
- process-local same-key FIFO queue
- optional auto refresh
- typed lock errors

## 核心設計

### 1. Owner Token

搶鎖時不只寫入固定值，而是寫入唯一 token。

釋放與續期時都要先確認 Redis key 目前的 value 仍等於自己的 token，只有一致才允許操作。

這可以避免：

- 誤刪其他 worker 後來取得的鎖
- 誤續期其他 worker 的鎖
- 鎖過期後舊 owner 仍然影響新 owner

### 2. Lua Release / Refresh

`Release` 和 `Refresh` 都需要以 Lua script 保證 compare-and-delete / compare-and-expire 的原子性。

`Release` 的語意：

- key 不存在：視為鎖已不在手上
- token 不一致：回傳 lock not held
- token 一致：刪除 key

`Refresh` 的語意：

- key 不存在：回傳 lock not held
- token 不一致：回傳 lock not held
- token 一致：更新 TTL

### 3. API 分層

`TryLock` 適合只嘗試一次的場景。

典型用途：

- cluster 中只允許一台跑的 cron
- housekeeping
- 可跳過的背景任務

`Acquire` 適合需要等待或重試的場景。

典型用途：

- 較長的臨界區
- 需要手動控制 release 時機
- 需要搭配 retry / backoff 的操作

`WithLock` 適合一般業務臨界區。

典型用途：

- 取得鎖
- 執行 callback
- 自動釋放
- 降低忘記 release 的風險

### 4. Same-Key FIFO Queue

同一個 process 內，對同一把 key 的 acquire 嘗試會先進本機 FIFO queue。

這樣可以：

- 減少同 process 內對 Redis 的無意義競爭
- 降低 hot key 壓力
- 讓同 key 的本機等待順序更可預測

這個 queue 只處理 process-local 公平性，不取代 Redis lock 本身的跨 process 協調。

### 5. Optional Auto Refresh

某些任務執行時間不穩定，鎖 TTL 不適合設得過大。

`Acquire` 可以選擇啟用 auto refresh，讓 lock 在持有期間定期續期。

使用原則：

- 短任務通常不需要 auto refresh
- 長任務可以啟用 auto refresh
- callback 結束後必須停止 refresh
- release 後不能再 refresh

### 6. Retry / Backoff

不同場景對等待鎖的要求不同，所以 retry strategy 不應寫死。

建議支援：

- 不重試
- 固定間隔重試
- 線性 backoff
- 最大次數限制
- context cancellation

### 7. Typed Errors

Lock manager 應該區分不同錯誤語意：

- key 為空或 TTL 無效
- 鎖未持有
- acquire timeout
- Redis backend error
- release / refresh error

這讓呼叫端可以依錯誤類型決定：

- 忽略
- 重試
- 回傳業務錯誤
- 記錄告警

## 使用原則

拿到鎖後不代表一定要執行昂貴操作。

在多 worker 競爭同一份共享狀態時，建議拿到鎖後再檢查一次狀態：

1. 嘗試取得鎖
2. 取得鎖後讀取最新共享狀態
3. 如果其他 worker 已經完成工作，直接返回
4. 如果仍需要處理，再執行實際工作

這可以減少重複計算與不必要的外部呼叫。

## 不建議的做法

### 各服務自行手寫 SetNX + Del

簡化版 `SetNX + Del` 通常缺少：

- owner token
- Lua release
- refresh
- retry / backoff
- typed errors
- same-key local queue

除非只是非常短暫且低風險的臨時保護，否則應優先使用 `RedisLockManager`。

### 把業務交易鎖混進 Redis lock manager

Redis lock manager 解的是短期分散式互斥。

資料一致性、DB transaction、unique constraint、idempotency table 仍應留在業務或資料層處理。

## 建議保留能力

`go-infra` 的 `RedisLockManager` 至少應保留：

1. `TryLock(ctx, key, ttl)`
2. `Acquire(ctx, key, ttl, opts...)`
3. `WithLock(ctx, key, ttl, opts, fn)`
4. owner token
5. Lua `Release`
6. Lua `Refresh`
7. retry / backoff options
8. typed lock errors
9. same-key process-local FIFO queue
10. optional auto refresh
