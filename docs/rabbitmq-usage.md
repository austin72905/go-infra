# RabbitMQ Usage

## 定位

`RabbitMQComponent` 負責 RabbitMQ 連線、publish、consume 與 lifecycle 整合。

底層套件使用：

```go
github.com/rabbitmq/amqp091-go
```

`go-infra` 只管理連線與常用 publish / subscribe 流程，不自動建立 exchange、queue、binding。拓撲宣告通常屬於服務自己的部署協議，建議由服務在初始化階段明確處理。

## Properties

```properties
rabbitmq.url=amqp://guest:guest@localhost:5672/
rabbitmq.exchange=app.events
rabbitmq.routing.key=events.created
rabbitmq.queue=app.events.created
rabbitmq.consumer.tag=
rabbitmq.prefetch=10
rabbitmq.auto.ack=false
rabbitmq.requeue=true
```

說明：

- `rabbitmq.url`：必填
- `rabbitmq.exchange`：publish 預設 exchange，可空字串
- `rabbitmq.routing.key`：`Publish(...)` 使用的預設 routing key
- `rabbitmq.queue`：`Subscribe(...)` 使用的預設 queue
- `rabbitmq.consumer.tag`：可選 consumer tag
- `rabbitmq.prefetch`：consumer QoS prefetch，`0` 代表不設定
- `rabbitmq.auto.ack`：預設 `false`
- `rabbitmq.requeue`：handler 回 error 時是否 nack requeue，預設 `false`

## Initialize

服務在 `Initialize()` 期間註冊 component：

```go
func (a *App) Initialize() {
	_ = a.LoadProperties(envProperties, "dev", "app.properties")

	a.RabbitMQ().LoadFromPrefix("rabbitmq")
}
```

named instance：

```go
a.RabbitMQ("analytics").LoadFromPrefix("rabbitmq.analytics")
```

## Publish

使用 properties 裡的 default exchange / routing key：

```go
err := a.RabbitMQ().Publish(ctx, []byte(`{"event":"created"}`))
```

指定 exchange / routing key：

```go
err := a.RabbitMQ().PublishTo(
	ctx,
	"app.events",
	"events.created",
	[]byte(`{"event":"created"}`),
)
```

設定 content type 與 persistent delivery：

```go
import "go-infra/rabbitmq"

err := a.RabbitMQ().Publish(
	ctx,
	body,
	rabbitmq.WithContentType("application/json"),
	rabbitmq.WithPersistentDelivery(),
)
```

## Subscribe

使用 properties 裡的 default queue：

```go
import "go-infra/rabbitmq"

a.RabbitMQ().Subscribe(func(ctx context.Context, msg rabbitmq.Message) error {
	// handler 回 nil 時 ack
	// handler 回 error 時 nack，是否 requeue 由 rabbitmq.requeue 決定
	return nil
})
```

指定 queue：

```go
a.RabbitMQ().SubscribeQueue("app.events.created", func(ctx context.Context, msg rabbitmq.Message) error {
	return nil
})
```

consumer 會在 startup lifecycle 的 Start 階段啟動，在 shutdown lifecycle 中先停止背景 consumer，再等待正在處理的 message 完成，最後關閉 RabbitMQ connection。

## Queue / Exchange Declaration

如果服務需要自行宣告 exchange、queue 或 binding，可以用底層 channel：

```go
ch, err := a.RabbitMQ().Client().Channel()
if err != nil {
	return err
}
defer ch.Close()

err = ch.ExchangeDeclare(
	"app.events",
	"topic",
	true,
	false,
	false,
	false,
	nil,
)
```

建議把 topology declaration 放在服務自己的初始化流程，避免共用 infra component 隱含建立業務拓撲。
