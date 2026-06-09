# gRPC Roadmap

這份文件整理 `go-infra` 目前 gRPC component 的能力邊界，以及下一步建議補齊的功能。

## Current Scope

目前 `go-infra` 已具備的 gRPC 能力：

- `a.Grpc()` component access
- `Listen(...)`
- `LoadFromPrefix(...)`
- `AddOpt(...)`
- `MaxConnections(...)`
- `Server() *grpc.Server`
- `HealthServer()`
- startup lifecycle 自動啟動 gRPC server
- shutdown lifecycle 自動 graceful stop

這一版定位是：

- 提供最小可用的 gRPC server infra
- 先把 component / lifecycle 模式收斂
- 讓服務端能維持接近 `a.Grpc().Server()` 的使用方式
- 暫時不追求和 `std-library` 同等完整度

## Priority 1

這些功能最值得先補，補完後實戰價值會明顯提升：

1. startup probe integration
   - 將 listen host / port 納入啟動前檢查
   - 讓 framework 在啟動期更一致

## Priority 2

這些功能值得保留，但可以等第一波穩定後再補：

1. prometheus interceptor / metrics
   - 讓 gRPC server 直接有基礎觀測能力

2. common interceptor helpers
   - 例如 logging / tracing / recovery 的統一入口

3. named gRPC instance support
   - 目前通常一個服務只會有一個 gRPC server
   - 如未來有需求，再考慮多 instance

## Not Urgent

這些能力原本框架有價值，但目前不建議太早搬進 `go-infra`：

1. 太多預設 interceptor 組合
2. 太厚的 shutdown phase 細分
3. metrics / logging / tracing 全家桶一次做滿
4. 太多 server tuning option 直接往 component 暴露

原因：

- 容易讓 `GrpcComponent` 重新膨脹
- 會過早綁死團隊對 gRPC middleware 的偏好
- 現階段更重要的是先把最小 server 能力做穩

## Suggested Target API

建議下一版 gRPC component 至少長成這樣：

- `Listen(listen string)`
- `LoadFromPrefix(prefix string)`
- `AddOpt(opt ...grpc.ServerOption) *GrpcComponent`
- `MaxConnections(max int32)`
- `Server() *grpc.Server`
- `HealthServer() *health.Server`
- `Start(ctx)`
- `Shutdown(ctx) error`

## Example Properties

```properties
grpc.listen=:50051
```

如果之後要再補更多設定，可往這種方向擴充：

```properties
grpc.listen=:50051
grpc.max.connections=1000
```
