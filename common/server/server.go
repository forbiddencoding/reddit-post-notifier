package server

import (
	"context"
	"errors"
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

func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		slog.Info("server shutdown signal received, starting graceful shutdown")

		if err := s.http.Shutdown(shutdownCtx); err != nil {
			slog.Error("http server graceful shutdown failed", slog.Any("error", err))
		}
	}()

	slog.Info("starting http server", "addr", s.http.Addr)
	err := s.http.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		slog.Info("http server shut down gracefully")
		return nil
	}

	return fmt.Errorf("could not start http server: %w", err)
}

func (s *Server) Close() error {
	slog.Info("closing http server immediately")
	return s.http.Close()
}
