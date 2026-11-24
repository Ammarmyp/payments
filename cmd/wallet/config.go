package main

import (
	"log/slog"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/config"
	"github.com/Ammarmyp/payments/internal/platform/logging"
)

// Config is everything the wallet service reads from its environment.
type Config struct {
	Env             string // development or production
	HTTPAddr        string
	ShutdownTimeout time.Duration
	LogLevel        slog.Level
	LogFormat       logging.Format
	DatabaseURL     string
	DBMaxConns      int
}

// loadConfig reads the configuration. Everything has a sensible default except
// secrets, and every missing or invalid variable is reported at once.
func loadConfig(env *config.Env) (Config, error) {
	c := Config{
		Env:             env.OneOf("APP_ENV", "development", "development", "production"),
		HTTPAddr:        env.String("WALLET_HTTP_ADDR", ":8081"),
		ShutdownTimeout: env.Duration("WALLET_SHUTDOWN_TIMEOUT", 10*time.Second),
		LogLevel:        config.Parse(env, "LOG_LEVEL", slog.LevelInfo, logging.ParseLevel),
		// Keep DATABASE_URL: shared with migrations and .env.example.
		DatabaseURL: env.Required("DATABASE_URL"),
		DBMaxConns:  env.Int("DATABASE_MAX_CONNS", 10),
	}

	// Text logs for humans in development, JSON for log collectors in production.
	format := logging.FormatText
	if c.Env == "production" {
		format = logging.FormatJSON
	}
	c.LogFormat = config.Parse(env, "LOG_FORMAT", format, logging.ParseFormat)

	if c.DBMaxConns < 1 || c.DBMaxConns > 100 {
		env.Invalid("DATABASE_MAX_CONNS", "must be between 1 and 100")
	}
	return c, env.Err()
}
