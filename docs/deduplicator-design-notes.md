# Deduplicator Design Notes

這份文件整理 `go-infra` 的 Deduplicator 設計原則。

## 結論

`Deduplicator` 的第一版定位應該是：

- 短期防重
- TTL-based
- Redis-backed
- API 薄且清楚
- 不承諾永久唯一性

它不應該一開始就變成事件總線框架、永久 idempotency 系統，或包含所有高吞吐優化的重型平台。

## 核心語意

Deduplicator 解的是「一段時間內，同一個 key 不要重複處理」。

典型場景：

- webhook callback 短期防重
- API 重複提交保護
- background job 重入保護
- event consumer 的短時間去重
- cron 任務避免短時間連續重跑

這類場景通常只需要：

1. key
2. TTL
3. mark-if-absent
4. optional check / clear

## Redis SetNX + TTL

第一版可以用 Redis `SetNX` 搭配 TTL 實作。

`TryMark(ctx, key, ttl)` 的語意：

- key 不存在時寫入 marker 並設定 TTL
- 寫入成功回傳 `true`
- key 已存在回傳 `false`
- Redis 失敗回傳 error

`Marked(ctx, key)` 的語意：

- key 存在回傳 `true`
- key 不存在回傳 `false`
- Redis 失敗回傳 error

`Clear(ctx, key)` 的語意：

- 刪除 marker
- 不要求 key 一定存在
- Redis 失敗回傳 error

## 短期防重不是永久唯一

Deduplicator 不應承諾永久唯一性。

原因：

- TTL 到期後 key 會消失
- Redis 可能短暫不可用
- Redis 資料可能因部署策略而被清除
- 不同服務可能使用不同 key 規則

永久唯一性應該由資料層保證，例如：

- DB unique key
- idempotency table
- event offset / sequence tracking
- business-level state machine

## 分層處理

建議把去重分成兩層：

1. `Deduplicator`：快速、短期、低成本
2. persistent idempotency：永久、最終一致、資料層保證

這樣可以讓 infra API 保持簡單，也避免把業務工作流放進通用套件。

## 高吞吐場景

高吞吐 event stream 可能會需要本地快篩，例如 Bloom Filter 或 in-memory window。

這類優化可以降低 Redis 存取量，但第一版不急著加入，因為它會引入額外參數：

- capacity
- false positive rate
- rotation window
- shard count
- local state lifecycle

建議等真的有明確流量需求後，再加入可選模式。

## 建議保留能力

第一版 `go-infra` 的 `Deduplicator` 建議保留：

1. `TryMark(ctx, key string, ttl time.Duration) (bool, error)`
2. `Marked(ctx, key string) (bool, error)`
3. `Clear(ctx, key string) error`
4. named Redis instance 綁定
5. typed errors
6. 清楚的 key 與 TTL 語意

## 不建議第一版加入

第一版暫不建議加入：

- 永久去重承諾
- DB fallback policy
- event bus 專用 API
- Bloom Filter 模式
- 批量 pipeline API
- 業務狀態機

這些都可以等需求穩定後再擴充。

## 使用原則

Deduplicator 適合回答：

**這個 key 在最近一段時間內是否已經處理過？**

它不適合回答：

**這件事在整個系統生命週期中是否永遠不會重複？**
