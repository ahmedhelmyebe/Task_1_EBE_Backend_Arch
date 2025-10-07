package config

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// InitRedis creates a single Redis client and verifies connectivity with Ping.
// It also configures sane timeouts so the app fails fast if Redis is unreachable.
func InitRedis(cfg *Config) *redis.Client {
	opts := &redis.Options{
		Addr:        cfg.RedisAddr,
		Password:    cfg.RedisPass,
		DB:          cfg.RedisDB,
		DialTimeout: 3 * time.Second,
		ReadTimeout: 2 * time.Second,
		WriteTimeout: 2 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	}
	rdb := redis.NewClient(opts)

	// verify connectivity (hard fail if Redis is down)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("[redis] ping failed: %v (addr=%s db=%d)", err, cfg.RedisAddr, cfg.RedisDB)
	}
	log.Printf("[redis] connected: addr=%s db=%d", cfg.RedisAddr, cfg.RedisDB)
	return rdb
}
