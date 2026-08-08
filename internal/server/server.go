package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgeirkou/tyrako/internal/config"
)

func New(cfg *config.Config, logger *slog.Logger, db *pgxpool.Pool) (*http.Server, func()) {
	router, cleanup := NewRouter(cfg, logger, db)
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}, cleanup
}
