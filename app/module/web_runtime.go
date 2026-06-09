package module

import (
	"context"
	"net/http"

	internalweb "go-infra/internal/web"

	"github.com/gin-gonic/gin"
)

type WebRuntime struct {
	engine *gin.Engine
	server *http.Server
	addr   string
}

func NewWebRuntime() *WebRuntime {
	engine := gin.New()
	engine.Use(
		internalweb.RequestContextMiddleware(),
		internalweb.RecoveryMiddleware(),
	)

	return &WebRuntime{
		engine: engine,
	}
}

func (w *WebRuntime) Router() *gin.Engine {
	return w.engine
}

func (w *WebRuntime) Listen(addr string) {
	w.addr = addr
}

func (w *WebRuntime) Addr() string {
	return w.addr
}

func (w *WebRuntime) Start() error {
	if w.server != nil {
		return nil
	}

	if w.addr == "" {
		w.addr = ":8080"
	}

	w.server = &http.Server{
		Addr:    w.addr,
		Handler: w.engine,
	}

	go func() {
		_ = w.server.ListenAndServe()
	}()

	return nil
}

func (w *WebRuntime) Shutdown(ctx context.Context) error {
	if w.server == nil {
		return nil
	}
	return w.server.Shutdown(ctx)
}
