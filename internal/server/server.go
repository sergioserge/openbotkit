package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/priyanshujain/openbotkit/config"
)

type Server struct {
	cfg  *config.Config
	addr string
}

func New(cfg *config.Config, addr string) *Server {
	return &Server{cfg: cfg, addr: addr}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	s.routes(mux)

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("server listening on %s", s.addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		log.Println("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
