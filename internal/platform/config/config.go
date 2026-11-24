// Package config reads typed environment values and reports every problem
// together instead of failing on the first bad key.
package config

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Env accumulates configuration problems while reading env values.
type Env struct {
	getenv   func(key string) (string, bool)
	problems []string
}

// New returns an Env that reads values through getenv.
func New(getenv func(key string) (string, bool)) *Env {
	return &Env{getenv: getenv}
}

func (e *Env) get(key string) (string, bool) {
	raw, _ := e.getenv(key)
	raw = strings.TrimSpace(raw)
	return raw, raw != ""
}

// Invalid records a configuration problem for key.
func (e *Env) Invalid(key, problem string) {
	e.problems = append(e.problems, key+": "+problem)
}

// Err returns one error listing every recorded problem, or nil.
func (e *Env) Err() error {
	if len(e.problems) == 0 {
		return nil
	}
	return fmt.Errorf("invalid configuration: %s", strings.Join(e.problems, "; "))
}

// String returns the env value for key, or def when unset or blank.
func (e *Env) String(key, def string) string {
	if v, ok := e.get(key); ok {
		return v
	}
	return def
}

// Required returns the env value for key, or records a problem when unset.
func (e *Env) Required(key string) string {
	v, ok := e.get(key)
	if !ok {
		e.Invalid(key, "required, but not set")
	}
	return v
}

// OneOf returns the env value for key (or def), and records a problem when
// the value is not in allowed.
func (e *Env) OneOf(key, def string, allowed ...string) string {
	v := e.String(key, def)
	if !slices.Contains(allowed, v) {
		e.Invalid(key, fmt.Sprintf("%q is not one of %s", v, strings.Join(allowed, ", ")))
	}
	return v
}

// Parse returns parse(env) for key, or def when unset. Invalid values are
// recorded as problems. It is a function because Go methods cannot have type
// parameters.
func Parse[T any](e *Env, key string, def T, parse func(string) (T, error)) T {
	v, ok := e.get(key)
	if !ok {
		return def
	}
	x, err := parse(v)
	if err != nil {
		e.Invalid(key, fmt.Sprintf("invalid value %q", v))
		return def
	}
	return x
}

// Int returns an int env value for key, or def when unset.
func (e *Env) Int(key string, def int) int {
	return Parse(e, key, def, strconv.Atoi)
}

// Duration returns a duration env value for key, or def when unset.
func (e *Env) Duration(key string, def time.Duration) time.Duration {
	return Parse(e, key, def, time.ParseDuration)
}
