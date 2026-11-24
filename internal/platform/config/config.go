// Package config reads typed environment values and reports every problem
// together instead of failing on the first bad key.
package config

import (
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Ammarmyp/payments/internal/platform/logging"
)

// Collector accumulates configuration problems while reading env values.
type Collector struct {
	getenv   func(key string) (string, bool)
	problems []string
}

// New returns a Collector that reads values through getenv.
func New(getenv func(key string) (string, bool)) *Collector {
	return &Collector{getenv: getenv}
}

func (c *Collector) get(key string) (string, bool) {
	raw, _ := c.getenv(key)
	raw = strings.TrimSpace(raw)
	return raw, raw != ""
}

// Invalid records a configuration problem for key.
func (c *Collector) Invalid(key, problem string) {
	c.problems = append(c.problems, key+": "+problem)
}

// LogConfig holds process log settings from the environment.
type LogConfig struct {
	Level  slog.Level
	Format logging.Format
}

// Err returns one error listing every recorded problem, or nil.
func (c *Collector) Err() error {
	if len(c.problems) == 0 {
		return nil
	}
	return fmt.Errorf("invalid configuration: %s", strings.Join(c.problems, "; "))
}

// String returns the env value for key, or def when unset or blank.
func (c *Collector) String(key, def string) string {
	if v, ok := c.get(key); ok {
		return v
	}
	return def
}

// Required returns the env value for key, or records a problem when unset.
func (c *Collector) Required(key string) string {
	v, ok := c.get(key)
	if !ok {
		c.Invalid(key, "required, but not set")
	}
	return v
}

// OneOf returns the env value for key (or def), and records a problem when
// the value is not in allowed.
func (c *Collector) OneOf(key, def string, allowed ...string) string {
	v := c.String(key, def)
	if !slices.Contains(allowed, v) {
		c.Invalid(key, fmt.Sprintf("%q is not one of %s", v, strings.Join(allowed, ", ")))
	}
	return v
}

// Parse returns parse(env) for key, or def when unset. Invalid values are
// recorded as problems. It is a function because Go methods cannot have type
// parameters.
func Parse[T any](c *Collector, key string, def T, parse func(string) (T, error)) T {
	v, ok := c.get(key)
	if !ok {
		return def
	}
	x, err := parse(v)
	if err != nil {
		c.Invalid(key, fmt.Sprintf("invalid value %q", v))
		return def
	}
	return x
}

// Int returns an int env value for key, or def when unset.
func (c *Collector) Int(key string, def int) int {
	return Parse(c, key, def, strconv.Atoi)
}

// Duration returns a duration env value for key, or def when unset.
func (c *Collector) Duration(key string, def time.Duration) time.Duration {
	return Parse(c, key, def, time.ParseDuration)
}
