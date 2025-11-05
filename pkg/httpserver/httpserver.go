package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Server wraps the HTTP server and provides start/stop functionality.
type Server struct {
	httpServer *http.Server
}

// New creates a new Server instance.
func New(addr string, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
		},
	}
}

// Start begins listening for HTTP requests.
func (s *Server) Start() {
	go func() {
		slog.Info("Starting HTTP server", "address", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	slog.Info("Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}
