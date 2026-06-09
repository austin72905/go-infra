package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"

	ogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	srv      *ogrpc.Server
	listen   string
	listener net.Listener
	health   *health.Server
	mu       sync.Mutex
	started  bool
}

func NewServer(opts ...ogrpc.ServerOption) *Server {
	srv := ogrpc.NewServer(opts...)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthServer)

	return &Server{
		srv:    srv,
		health: healthServer,
	}
}

func (s *Server) SetListen(listen string) {
	s.listen = listen
}

func (s *Server) Listen() string {
	return s.listen
}

func (s *Server) GRPCServer() *ogrpc.Server {
	return s.srv
}

func (s *Server) HealthServer() *health.Server {
	return s.health
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}
	if s.listen == "" {
		return fmt.Errorf("grpc listen is required")
	}

	listener, err := net.Listen("tcp", s.listen)
	if err != nil {
		return err
	}

	s.listener = listener
	s.started = true
	s.health.Resume()
	go func() {
		_ = s.srv.Serve(listener)
	}()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.health.Shutdown()
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.srv.GracefulStop()
	}()

	select {
	case <-ctx.Done():
		s.srv.Stop()
		return ctx.Err()
	case <-done:
		s.mu.Lock()
		s.started = false
		s.listener = nil
		s.mu.Unlock()
		return nil
	}
}
