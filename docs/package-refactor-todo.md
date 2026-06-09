# go-infra Package Refactor TODO

## 1. 目的

目前 `app/module` 同時承擔了：

- app lifecycle
- web runtime
- property loading
- startup probe
- db
- redis
- rabbitmq
- kafka
- grpc
- mongo
- scheduler

這會導致外部專案只要 import `github.com/austin72905/go-infra/app/module`，
Go 就會一起編譯整個 package 內所有 `.go` 檔案。

結果是：

- 即使外部專案只想用 web runtime
- 也會一起編譯 kafka / redis / db / rabbitmq 等依賴
- 造成不必要耦合與編譯失敗風險

重構目標是：

- 把不同基礎設施拆成小 package
- 讓外部專案只引用自己需要的能力
- 避免 `app/module` 成為大而全 package

---

## 2. 重構原則

### 2.1 一個 package 只負責一類能力

例如：

- `web` 只管 web server
- `config` 只管 properties
- `probe` 只管 startup probe
- `db` 只管 db component
- `redis` 只管 redis 能力
- `rabbitmq` 只管 rabbitmq 能力
- `kafka` 只管 kafka 能力

### 2.2 避免巨型入口 package

不要再維持：

- `ModuleBase.Web()`
- `ModuleBase.DB()`
- `ModuleBase.Redis()`
- `ModuleBase.Kafka()`

這種全能力入口會重新把 package 黏回一起。

### 2.3 外部專案應以組合取代全包入口

外部使用方式應該接近：

```go
runtime := app.NewRuntime()
webRuntime := web.New()
props := config.New()
probe := probe.New()
```

而不是：

```go
m.Web()
m.DB()
m.Redis()
m.Kafka()
```

---

## 3. 目標 package 結構

```text
go-infra/
  app/
    app.go
    runtime.go
    lifecycle.go
    component.go
    component_registry.go

  web/
    runtime.go

  config/
    property.go

  probe/
    startup.go

  db/
    component.go

  redis/
    component.go
    lock.go
    dedupe.go

  rabbitmq/
    client.go
    component.go

  kafka/
    client.go
    component.go

  grpc/
    server_component.go
    client_component.go

  mongo/
    component.go

  scheduler/
    component.go
```

---

## 4. 檔案搬移對照

### 4.1 第一批：先拆可獨立使用的核心包

- `app/module/app.go` -> `app/app.go`
- `app/module/app_runtime.go` -> `app/runtime.go`
- `app/module/lifecycle.go` -> `app/lifecycle.go`
- `app/module/component.go` -> `app/component.go`
- `app/module/component_registry.go` -> `app/component_registry.go`
- `app/module/web_runtime.go` -> `web/runtime.go`
- `app/module/property_runtime.go` -> `config/property.go`
- `app/module/startup_probe.go` -> `probe/startup.go`

### 4.2 第二批：infra 能力包

- `app/module/db_component.go` -> `db/component.go`
- `app/module/redis_component.go` -> `redis/component.go`
- `app/module/redis_lock_manager.go` -> `redis/lock.go`
- `app/module/deduplicator.go` -> `redis/dedupe.go`
- `app/module/rabbitmq_component.go` -> `rabbitmq/component.go`
- `app/module/kafka_component.go` -> `kafka/component.go`
- `app/module/grpc_component.go` -> `grpc/server_component.go`
- `app/module/grpc_client_component.go` -> `grpc/client_component.go`
- `app/module/mongo_component.go` -> `mongo/component.go`
- `app/module/scheduler_component.go` -> `scheduler/component.go`

### 4.3 第三批：刪除或重寫巨型入口

- `app/module/module_base.go`

這個檔案不建議原封不動保留，應拆解或重寫。

---

## 5. package name 調整

搬檔案時，必須同時修改 package name。

例如：

- `web/runtime.go` -> `package web`
- `config/property.go` -> `package config`
- `probe/startup.go` -> `package probe`
- `db/component.go` -> `package db`
- `redis/lock.go` -> `package redis`
- `rabbitmq/component.go` -> `package rabbitmq`
- `kafka/component.go` -> `package kafka`

只搬資料夾、不改 package name 沒有意義。

---

## 6. `ModuleBase` 重構策略

### 現況問題

`ModuleBase` 現在同時暴露：

- `Web()`
- `DB()`
- `Redis()`
- `Kafka()`
- `RabbitMQ()`
- `Mongo()`
- `Grpc()`
- `Schedule()`

這會讓 `app/module` 永遠維持高度耦合。

### 建議處理方式

不要保留原本全能力版本。

改成：

- `app` 只保留 runtime / lifecycle / registry
- 其他 infra package 自己負責初始化
- 外部專案自己組裝

例如：

```go
runtime := app.NewRuntime()
webRuntime := web.New()
runtime.AttachWeb(webRuntime)
```

如果之後真的要有 helper，也要做成小而明確的 helper，而不是全包式 `ModuleBase`。

---

## 7. 建議執行順序

### Step 1

先拆：

- `web`
- `config`
- `probe`
- `app`

目標：

- 讓外部專案可以安全 import `app` + `web`
- 不再把 kafka 等能力一起編譯進來

### Step 2

調整 `app` 對外 API：

- `Start(...)`
- `Runtime`
- `Lifecycle`

移除對 DB / Redis / Kafka 的直接耦合。

### Step 3

再拆：

- `db`
- `redis`
- `rabbitmq`
- `kafka`
- `grpc`
- `mongo`
- `scheduler`

### Step 4

更新文件與範例：

- `overview.md`
- 各 capability 的 usage 文件
- 外部專案整合示例

---

## 8. 驗收標準

重構完成後，應滿足：

1. 外部專案 import `github.com/austin72905/go-infra/app` 不會編譯 kafka
2. 外部專案 import `github.com/austin72905/go-infra/web` 不會編譯 db / redis / kafka
3. `rabbitmq`、`redis`、`kafka` 能單獨被引用
4. `go test ./...` 可以通過
5. 文件能清楚說明各 package 的用途

---

## 9. 第一批實作建議

如果下一步要真的動手改，建議優先做：

1. 建立 `web/` package
2. 建立 `config/` package
3. 建立 `probe/` package
4. 建立 `app/` package
5. 讓一個外部 demo 專案只用 `app + web`

等這個流程穩了，再繼續拆 infra 能力包。
