package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/mookle/fennel/apps/product/internal/httpapi"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := loadConfig([]string{
		"PATH=/usr/bin",
		"FENNEL_PRODUCT_DSN=postgres://product@db/product",
		"FENNEL_PRODUCT_TOKEN_CART=cart-token",
		"FENNEL_PRODUCT_TOKEN_DEV=dev=token",
	})
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if cfg.DSN != "postgres://product@db/product" {
		t.Errorf("DSN = %q", cfg.DSN)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want the default :8080", cfg.Addr)
	}

	// A token can hold an equals sign, because only the first one splits
	// the name from the value.
	want := []httpapi.Caller{
		{Name: "cart", Token: "cart-token"},
		{Name: "dev", Token: "dev=token"},
	}
	if !slices.Equal(cfg.Callers, want) {
		t.Errorf("Callers = %v, want %v", cfg.Callers, want)
	}
}

func TestLoadConfigReadsTheAddress(t *testing.T) {
	cfg, err := loadConfig([]string{
		"FENNEL_PRODUCT_DSN=postgres://product@db/product",
		"FENNEL_PRODUCT_ADDR=127.0.0.1:9000",
		"FENNEL_PRODUCT_TOKEN_CART=cart-token",
	})
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Addr != "127.0.0.1:9000" {
		t.Errorf("Addr = %q, want 127.0.0.1:9000", cfg.Addr)
	}
}

func TestLoadConfigRejects(t *testing.T) {
	const dsn = "FENNEL_PRODUCT_DSN=postgres://product@db/product"

	cases := []struct {
		name    string
		environ []string
		want    []string
	}{
		{
			name:    "an unset DSN",
			environ: []string{"FENNEL_PRODUCT_TOKEN_CART=cart-token"},
			want:    []string{"FENNEL_PRODUCT_DSN is unset"},
		},
		{
			name:    "no caller at all",
			environ: []string{dsn},
			want:    []string{"no FENNEL_PRODUCT_TOKEN_* variable is set"},
		},
		{
			name:    "an empty token",
			environ: []string{dsn, "FENNEL_PRODUCT_TOKEN_CART="},
			want:    []string{"FENNEL_PRODUCT_TOKEN_CART is empty"},
		},
		{
			name:    "a variable that names no caller",
			environ: []string{dsn, "FENNEL_PRODUCT_TOKEN_=a-token"},
			want:    []string{"FENNEL_PRODUCT_TOKEN_ names no caller"},
		},
		{
			name: "two callers that share a token",
			environ: []string{dsn,
				"FENNEL_PRODUCT_TOKEN_CART=same", "FENNEL_PRODUCT_TOKEN_DEV=same"},
			want: []string{`FENNEL_PRODUCT_TOKEN_DEV holds the token of caller "cart"`},
		},
		{
			// Every problem reports at once, so a broken environment takes
			// one restart to fix rather than one per variable.
			name:    "every problem at once",
			environ: []string{"FENNEL_PRODUCT_TOKEN_CART="},
			want:    []string{"FENNEL_PRODUCT_TOKEN_CART is empty", "FENNEL_PRODUCT_DSN is unset"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := loadConfig(c.environ)
			if err == nil {
				t.Fatal("loadConfig succeeded, want an error")
			}
			for _, w := range c.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not mention %q", err, w)
				}
			}
		})
	}
}
