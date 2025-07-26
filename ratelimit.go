package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	RedisClient *redis.Client
	MaxRequests int
	Duration    time.Duration
}

type Limiter struct {
	config Config
}

func NewLimiter(config Config) *Limiter {
	return &Limiter{
		config: config,
	}
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		clientAddr := getClientAddr(r.Header)
		if clientAddr == "" {
			next.ServeHTTP(w, r)
			return
		}

		key := fmt.Sprintf("httprate:%s", clientAddr)

		current, err := l.config.RedisClient.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			next.ServeHTTP(w, r)
			return
		}

		if current >= l.config.MaxRequests {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		tx := l.config.RedisClient.TxPipeline()

		_ = tx.Incr(ctx, key).Err()

		if current == 0 {
			_ = tx.Expire(ctx, key, l.config.Duration).Err()
		}

		_, err = tx.Exec(ctx)

		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) MiddlewareWithSlog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		reqID := r.Header.Get("X-Request-Id")
		logger := slog.With("request_id", reqID)

		clientAddr := getClientAddr(r.Header)
		if clientAddr == "" {
			logger.Error("Failed to extract client IP")
			next.ServeHTTP(w, r)
			return
		}

		key := fmt.Sprintf("httprate:%s", clientAddr)
		logger = logger.With("key", key)

		current, err := l.config.RedisClient.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			logger.Error("Redis failed get value", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		if current >= l.config.MaxRequests {
			logger.Warn("Rate limit exceeded")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		tx := l.config.RedisClient.TxPipeline()

		if err = tx.Incr(ctx, key).Err(); err != nil {
			logger.Error("Redis failed incr value", "error", err)
		}

		if current == 0 {
			if err = tx.Expire(ctx, key, l.config.Duration).Err(); err != nil {
				logger.Error("Redis failed set TTL", "error", err)
			}
		}

		if _, err = tx.Exec(ctx); err != nil {
			logger.Error("Redis failed execute", "error", err)
		}

		next.ServeHTTP(w, r)
	})
}

func getClientAddr(header http.Header) string {
	if clientAddr := header.Get("Cf-Connecting-Ip"); clientAddr != "" {
		return clientAddr
	}

	if clientAddr := header.Get("X-Forwarded-For"); clientAddr != "" {
		return clientAddr
	}

	return ""
}
