// Package postgres connects services to PostgreSQL through a pgx connection pool.
package postgres

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds PostgreSQL pool settings. Use DefaultConfig for conservative
// production-friendly defaults, then override fields or load from the environment.
type Config struct {
	URL string

	// Pool sizing: a few warm connections, and a hard cap well below
	// Postgres's max_connections, so many replicas can share one server.
	MinConns int32
	MaxConns int32

	// Recycle connections so load rebalances after a failover.
	// MaxConnLifetimeJitter stops every connection from reconnecting at once.
	MaxConnLifetime       time.Duration
	MaxConnLifetimeJitter time.Duration
	MaxConnIdleTime       time.Duration

	HealthCheckPeriod  time.Duration
	ConnectTimeout     time.Duration
	StartupPingTimeout time.Duration
}

// DefaultConfig returns conservative pool tuning for the given database URL.
func DefaultConfig(url string) Config {
	return Config{
		URL:                   url,
		MinConns:              2,
		MaxConns:              10,
		MaxConnLifetime:       time.Hour,
		MaxConnLifetimeJitter: 5 * time.Minute,
		MaxConnIdleTime:       15 * time.Minute,
		HealthCheckPeriod:     30 * time.Second,
		ConnectTimeout:        5 * time.Second,
		StartupPingTimeout:    5 * time.Second,
	}
}

// ConfigFromEnv reads DATABASE_URL (required) and optional pool overrides.
//
// Optional env:
//   - DATABASE_MAX_CONNS
//   - DATABASE_MIN_CONNS
//   - DATABASE_MAX_CONN_LIFETIME
//   - DATABASE_MAX_CONN_LIFETIME_JITTER
//   - DATABASE_MAX_CONN_IDLE_TIME
//   - DATABASE_HEALTH_CHECK_PERIOD
//   - DATABASE_CONNECT_TIMEOUT
//   - DATABASE_STARTUP_PING_TIMEOUT
func ConfigFromEnv() (Config, error) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return Config{}, fmt.Errorf("postgres: DATABASE_URL is required")
	}

	cfg := DefaultConfig(url)

	if v, ok, err := envInt32("DATABASE_MAX_CONNS"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.MaxConns = v
	}
	if v, ok, err := envInt32("DATABASE_MIN_CONNS"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.MinConns = v
	}
	if v, ok, err := envDuration("DATABASE_MAX_CONN_LIFETIME"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.MaxConnLifetime = v
	}
	if v, ok, err := envDuration("DATABASE_MAX_CONN_LIFETIME_JITTER"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.MaxConnLifetimeJitter = v
	}
	if v, ok, err := envDuration("DATABASE_MAX_CONN_IDLE_TIME"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.MaxConnIdleTime = v
	}
	if v, ok, err := envDuration("DATABASE_HEALTH_CHECK_PERIOD"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.HealthCheckPeriod = v
	}
	if v, ok, err := envDuration("DATABASE_CONNECT_TIMEOUT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.ConnectTimeout = v
	}
	if v, ok, err := envDuration("DATABASE_STARTUP_PING_TIMEOUT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.StartupPingTimeout = v
	}

	return cfg, nil
}

// Open creates a pool, pings with a bounded timeout, and closes the pool on ping failure.
// The pool connects lazily; pinging here fails startup on a bad DSN or unreachable database
// instead of on the first request.
func Open(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse database url: %w", err)
	}

	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnLifetimeJitter = cfg.MaxConnLifetimeJitter
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	pingTimeout := cfg.StartupPingTimeout
	if pingTimeout <= 0 {
		pingTimeout = DefaultConfig(cfg.URL).StartupPingTimeout
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	return pool, nil
}

// NewPool opens a pool for databaseURL with the given max connections and
// otherwise conservative defaults.
func NewPool(ctx context.Context, databaseURL string, maxConns int32) (*pgxpool.Pool, error) {
	cfg := DefaultConfig(databaseURL)
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	return Open(ctx, cfg)
}

func envInt32(key string) (int32, bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return 0, false, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, false, fmt.Errorf("postgres: parse %s: %w", key, err)
	}
	return int32(n), true, nil
}

func envDuration(key string) (time.Duration, bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return 0, false, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, false, fmt.Errorf("postgres: parse %s: %w", key, err)
	}
	return d, true, nil
}
