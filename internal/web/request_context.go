package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	headerRequestID     = "X-Request-Id"
	headerCorrelationID = "X-Correlation-Id"
	headerTrace         = "X-Trace"
)

type contextKey string

const (
	requestIDContextKey     contextKey = "requestID"
	correlationIDContextKey contextKey = "correlationID"
	traceContextKey         contextKey = "trace"
	clientIPContextKey      contextKey = "clientIP"
	actionContextKey        contextKey = "action"
)

type RequestMetadata struct {
	RequestID     string
	CorrelationID string
	Trace         bool
	ClientIP      string
	Action        string
}

func RequestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		metadata := ParseRequestMetadata(c.Request, c.FullPath(), c.ClientIP())
		ctx := WithRequestMetadata(c.Request.Context(), metadata)
		c.Request = c.Request.WithContext(ctx)

		if metadata.RequestID != "" {
			c.Header(headerRequestID, metadata.RequestID)
		}

		c.Next()
	}
}

func ParseRequestMetadata(r *http.Request, fullPath, clientIP string) RequestMetadata {
	requestID := strings.TrimSpace(r.Header.Get(headerRequestID))
	if requestID == "" {
		requestID = newRequestID()
	}

	correlationID := strings.TrimSpace(r.Header.Get(headerCorrelationID))
	trace := strings.EqualFold(strings.TrimSpace(r.Header.Get(headerTrace)), "true")
	action := parseAction(r.Method, fullPath, r.URL.Path)

	return RequestMetadata{
		RequestID:     requestID,
		CorrelationID: correlationID,
		Trace:         trace,
		ClientIP:      clientIP,
		Action:        action,
	}
}

func WithRequestMetadata(ctx context.Context, metadata RequestMetadata) context.Context {
	ctx = context.WithValue(ctx, requestIDContextKey, metadata.RequestID)
	ctx = context.WithValue(ctx, correlationIDContextKey, metadata.CorrelationID)
	ctx = context.WithValue(ctx, traceContextKey, metadata.Trace)
	ctx = context.WithValue(ctx, clientIPContextKey, metadata.ClientIP)
	ctx = context.WithValue(ctx, actionContextKey, metadata.Action)
	return ctx
}

func RequestID(ctx context.Context) string {
	return stringValue(ctx, requestIDContextKey)
}

func CorrelationID(ctx context.Context) string {
	return stringValue(ctx, correlationIDContextKey)
}

func TraceEnabled(ctx context.Context) bool {
	trace, _ := ctx.Value(traceContextKey).(bool)
	return trace
}

func ClientIP(ctx context.Context) string {
	return stringValue(ctx, clientIPContextKey)
}

func Action(ctx context.Context) string {
	return stringValue(ctx, actionContextKey)
}

func stringValue(ctx context.Context, key contextKey) string {
	value, _ := ctx.Value(key).(string)
	return value
}

func parseAction(method, fullPath, requestPath string) string {
	actionPath := fullPath
	if actionPath == "" {
		actionPath = requestPath
	}
	return "http:" + strings.ToLower(method) + ":" + actionPath
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}
