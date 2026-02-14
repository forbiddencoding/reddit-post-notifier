package server

import (
	"context"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	http *http.Server
}

func New(handler http.Handler, config *config.Config) *Server {
	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler: handler,
	}

	s.RegisterOnShutdown(func() {
		s.SetKeepAlivesEnabled(false)
	})

	return &Server{http: s}
}

func (s *Server) Start() error {
	slog.Info("starting http server", "addr", s.http.Addr)
	err := s.http.ListenAndServe()

	return fmt.Errorf("could not start http server: %w", err)
}

func (s *Server) Close() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.Info("server shutdown signal received, starting graceful shutdown")

	if err := s.http.Shutdown(shutdownCtx); err != nil {
		slog.Warn("http server graceful shutdown failed, forcing close", slog.Any("error", err))
		return s.http.Close()
	}

	slog.Info("http server shutdown complete")
	return s.http.Close()
}
