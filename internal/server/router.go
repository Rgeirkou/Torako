package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/rgeirkou/tyrako/internal/config"
	"github.com/rgeirkou/tyrako/internal/handler"
	custommw "github.com/rgeirkou/tyrako/internal/middleware"
	"github.com/rgeirkou/tyrako/internal/model"
	"github.com/rgeirkou/tyrako/internal/provider/truemoney"
	"github.com/rgeirkou/tyrako/internal/ratelimit"
	"github.com/rgeirkou/tyrako/internal/ratelimit/redislimit"
	"github.com/rgeirkou/tyrako/internal/repository/memory"
	"github.com/rgeirkou/tyrako/internal/repository/postgres"
	"github.com/rgeirkou/tyrako/internal/service"
)

type apiKeyStore interface {
	Create(ctx context.Context, key *model.ApiKey) error
	GetByHash(ctx context.Context, hash string) (*model.ApiKey, error)
	List(ctx context.Context) ([]model.ApiKey, error)
	Revoke(ctx context.Context, id int64) error
	Touch(ctx context.Context, id int64) error
}

type statsStore interface {
	RecordTw(ctx context.Context, amountCents int64, ref string) error
	RecordTwError(ctx context.Context) error
	Totals(ctx context.Context) (*model.Stats, error)
}

func NewRouter(cfg *config.Config, logger *slog.Logger, db *pgxpool.Pool) (http.Handler, func()) {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(custommw.Logging(logger))
	r.Use(custommw.Recover(logger))
	r.Use(custommw.CORS(cfg.AllowOrigins))
	r.Use(custommw.SecurityHeaders)
	r.Use(chimw.Timeout(cfg.WriteTimeout))

	var redisClient *redis.Client
	var memberLimiter, partnerLimiter, ipRateLimiter ratelimit.Limiter
	if cfg.RedisURL != "" {
		opts, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			logger.Error("invalid REDIS_URL", "err", err)
		} else {
			redisClient = redis.NewClient(opts)
		}
	}
	if cfg.RateLimitEnabled {
		if redisClient != nil {
			memberLimiter = redislimit.New(redisClient, cfg.RateLimitMemberMax, cfg.RateLimitWindow)
			partnerLimiter = redislimit.New(redisClient, cfg.RateLimitPartnerMax, cfg.RateLimitWindow)
		} else {
			memberLimiter = ratelimit.NewMemory(cfg.RateLimitMemberMax, cfg.RateLimitWindow)
			partnerLimiter = ratelimit.NewMemory(cfg.RateLimitPartnerMax, cfg.RateLimitWindow)
		}
	}
	if cfg.RateLimitIPEnabled {
		if redisClient != nil {
			ipRateLimiter = redislimit.New(redisClient, cfg.RateLimitIPMax, cfg.RateLimitIPWindow)
		} else {
			ipRateLimiter = ratelimit.NewMemory(cfg.RateLimitIPMax, cfg.RateLimitIPWindow)
		}
	}
	cleanup := func() {
		if redisClient != nil {
			_ = redisClient.Close()
		}
	}

	tmClient := truemoney.NewClient(cfg.TrueMoneyBaseURL, cfg.TrueMoneyTimeout, logger)

	var keyStore apiKeyStore = memory.NewKeyStore()
	if db != nil {
		keyStore = postgres.NewKeyStore(db)
	}
	keySvc := service.NewKeyService(keyStore)
	keyHandler := handler.NewKeyHandler(keySvc)

	var statsStore statsStore = memory.NewStatsStore()
	if db != nil {
		statsStore = postgres.NewStatsStore(db)
	}
	statsHandler := handler.NewStatsHandler(statsStore)

	twSvc := service.NewTwService(service.NewCoalescedTwProvider(tmClient), statsStore, logger)
	twHandler := handler.NewTwHandler(twSvc)

	bootstrapCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := keySvc.Bootstrap(bootstrapCtx, cfg.BootstrapAPIKey, logger); err != nil {
		logger.Error("bootstrap api key", "err", err)
	}

	r.Route("/tw", func(api chi.Router) {
		if ipRateLimiter != nil {
			api.Use(custommw.RateLimitIP(ipRateLimiter, cfg.TrustProxyHeaders))
		}
		api.Use(custommw.Auth(keyStore, logger))
		if memberLimiter != nil {
			api.Use(custommw.RateLimit(memberLimiter, partnerLimiter, cfg.TrustProxyHeaders))
		}
		api.Get("/{phone}/{gift:*}", twHandler.RedeemGet)
		api.Post("/", twHandler.RedeemPost)
	})

	r.Route("/keys", func(api chi.Router) {
		if ipRateLimiter != nil {
			api.Use(custommw.RateLimitIP(ipRateLimiter, cfg.TrustProxyHeaders))
		}
		api.Use(custommw.Auth(keyStore, logger))
		if memberLimiter != nil {
			api.Use(custommw.RateLimit(memberLimiter, partnerLimiter, cfg.TrustProxyHeaders))
		}
		api.Use(custommw.RequireScope("admin"))
		api.Post("/", keyHandler.Create)
		api.Get("/", keyHandler.List)
		api.Delete("/{id}", keyHandler.Revoke)
		api.Post("/{id}/rotate", keyHandler.Rotate)
	})

	r.Route("/stats", func(api chi.Router) {
		if ipRateLimiter != nil {
			api.Use(custommw.RateLimitIP(ipRateLimiter, cfg.TrustProxyHeaders))
		}
		api.Get("/", statsHandler.Stats)
	})

	return r, cleanup
}
