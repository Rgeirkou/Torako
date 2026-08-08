package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgeirkou/tyrako/internal/config"
	"github.com/rgeirkou/tyrako/internal/repository/postgres"
	"github.com/rgeirkou/tyrako/internal/server"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config failed", "err", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var db *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		db, err = postgres.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			logger.Error("connect db failed", "err", err)
			return 1
		}
		defer db.Close()
		if err := postgres.Migrate(ctx, db); err != nil {
			logger.Error("migrate db failed", "err", err)
			return 1
		}
	} else {
		logger.Warn("DATABASE_URL not set, running without database")
	}

	srv, cleanup := server.New(cfg, logger, db)
	defer cleanup()
	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", cfg.Addr, "env", cfg.Env)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		logger.Error("server error", "err", err)
		return 1
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	}
	logger.Info("server stopped")
	return 0
}
