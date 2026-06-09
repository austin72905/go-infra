package web

import (
    "context"
    "net/http"

    internalweb "github.com/austin72905/go-infra/internal/web"
    "github.com/gin-gonic/gin"
)

type Runtime struct {
    engine *gin.Engine
    server *http.Server
    addr   string
}

func New() *Runtime {
    engine := gin.New()
    engine.Use(
        internalweb.RequestContextMiddleware(),
        internalweb.RecoveryMiddleware(),
    )

    return &Runtime{engine: engine}
}

func (w *Runtime) Router() *gin.Engine { return w.engine }
func (w *Runtime) Listen(addr string) { w.addr = addr }
func (w *Runtime) Addr() string { return w.addr }

func (w *Runtime) Start() error {
    if w.server != nil {
        return nil
    }
    if w.addr == "" {
        w.addr = ":8080"
    }
    w.server = &http.Server{Addr: w.addr, Handler: w.engine}
    go func() { _ = w.server.ListenAndServe() }()
    return nil
}

func (w *Runtime) Shutdown(ctx context.Context) error {
    if w.server == nil {
        return nil
    }
    return w.server.Shutdown(ctx)
}
