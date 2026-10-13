package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mookle/fennel/apps/product/internal/httpapi"
)

// The environment variables the service reads. Compose sets them (ADR-0033).
const (
	// dsnEnv points the service at its own Postgres (ADR-0002). The store
	// tests read the same variable.
	dsnEnv = "FENNEL_PRODUCT_DSN"

	// addrEnv is the address the HTTP server listens on. The contract's
	// server URL names port 8080, so that is the default.
	addrEnv     = "FENNEL_PRODUCT_ADDR"
	defaultAddr = ":8080"

	// callerTokenPrefix starts the name of every caller's token variable.
	// The rest of the name names the caller, so FENNEL_PRODUCT_TOKEN_CART
	// holds the token cart sends. A new caller is a new variable and no
	// code change (ADR-0033).
	callerTokenPrefix = "FENNEL_PRODUCT_TOKEN_"
)

// config is everything the service reads from its environment. main reads
// it once, before the server starts, so changing a value takes a restart.
type config struct {
	DSN     string
	Addr    string
	Callers []httpapi.Caller
}

// loadConfig reads the configuration from environ, which takes the form of
// os.Environ, so a test can pass a literal list.
func loadConfig(environ []string) (config, error) {
	cfg := config{Addr: defaultAddr}
	var errs []error

	for _, kv := range environ {
		key, value, _ := strings.Cut(kv, "=")
		switch {
		case key == dsnEnv:
			cfg.DSN = value
		case key == addrEnv && value != "":
			cfg.Addr = value
		case strings.HasPrefix(key, callerTokenPrefix):
			c, err := parseCaller(key, value, cfg.Callers)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			cfg.Callers = append(cfg.Callers, c)
		}
	}

	if cfg.DSN == "" {
		errs = append(errs, fmt.Errorf("%s is unset", dsnEnv))
	}
	if len(cfg.Callers) == 0 && len(errs) == 0 {
		errs = append(errs, fmt.Errorf("no %s* variable is set, so no caller can authenticate", callerTokenPrefix))
	}
	if err := errors.Join(errs...); err != nil {
		return config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// parseCaller reads one caller's token variable. The name lowercases, so
// FENNEL_PRODUCT_TOKEN_CART is the caller "cart". It rejects an empty name
// or token, and a token that an earlier caller already holds, because two
// callers that share a token cannot be told apart.
func parseCaller(key, token string, earlier []httpapi.Caller) (httpapi.Caller, error) {
	name := strings.ToLower(strings.TrimPrefix(key, callerTokenPrefix))
	if name == "" {
		return httpapi.Caller{}, fmt.Errorf("%s names no caller", key)
	}
	if token == "" {
		return httpapi.Caller{}, fmt.Errorf("%s is empty", key)
	}
	for _, c := range earlier {
		if c.Token == token {
			return httpapi.Caller{}, fmt.Errorf("%s holds the token of caller %q", key, c.Name)
		}
	}
	return httpapi.Caller{Name: name, Token: token}, nil
}
