package grpc

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	ogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	DefaultConnPoolSize = 3
)

type ClientPool struct {
	conns   []*ogrpc.ClientConn
	counter atomic.Uint64
}

func NewClientPool(address string, connPoolSize int, opt ...ogrpc.DialOption) *ClientPool {
	if address == "" {
		panic("grpc client address is required")
	}
	if connPoolSize <= 0 {
		connPoolSize = 1
	}

	conns := make([]*ogrpc.ClientConn, connPoolSize)
	for i := 0; i < connPoolSize; i++ {
		conns[i] = dial(address, opt...)
	}

	return &ClientPool{conns: conns}
}

func (p *ClientPool) Get() *ogrpc.ClientConn {
	if len(p.conns) == 0 {
		panic("grpc client pool is empty")
	}
	idx := p.counter.Add(1) % uint64(len(p.conns))
	return p.conns[idx]
}

func (p *ClientPool) Close() error {
	var firstErr error
	for _, conn := range p.conns {
		if conn == nil {
			continue
		}
		if err := conn.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func dial(address string, opt ...ogrpc.DialOption) *ogrpc.ClientConn {
	options := []ogrpc.DialOption{
		ogrpc.WithTransportCredentials(insecure.NewCredentials()),
		ogrpc.WithConnectParams(ogrpc.ConnectParams{
			Backoff:           backoff.DefaultConfig,
			MinConnectTimeout: 5 * time.Second,
		}),
		ogrpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	if len(opt) > 0 {
		options = append(options, opt...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := ogrpc.DialContext(ctx, address, options...)
	if err != nil {
		panic(fmt.Sprintf("dial grpc client %q failed: %v", address, err))
	}
	return conn
}
