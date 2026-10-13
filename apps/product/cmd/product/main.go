// Command product serves the product catalogue over REST.
//
// It reads its configuration from the environment, migrates its own
// database, and serves until SIGINT or SIGTERM, when it drains the requests
// in flight before it exits.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/mookle/fennel/apps/product/internal/catalogue"
	"github.com/mookle/fennel/apps/product/internal/httpapi"
	"github.com/mookle/fennel/apps/product/migrations"
)

const (
	// startupTimeout bounds the database ping and the migrations, so a
	// database that never answers fails the start rather than hanging it.
	startupTimeout = 30 * time.Second

	// shutdownTimeout bounds the drain. Compose sends SIGKILL ten seconds
	// after SIGTERM by default, so the drain finishes inside that.
	shutdownTimeout = 8 * time.Second
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("product: exit", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig(os.Environ())
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := migrate(ctx, db); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.RequireBearer(cfg.Callers, httpapi.New(catalogue.NewStore(db))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// The names show which callers the environment configured, so a
	// misspelt variable shows up here rather than as a run of 401s. The
	// tokens never reach a log.
	names := make([]string, len(cfg.Callers))
	for i, c := range cfg.Callers {
		names[i] = c.Name
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("product: listening", "addr", cfg.Addr, "callers", names)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	// A second signal during the drain kills the process at once.
	stop()
	slog.Info("product: shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// migrate brings the schema up to date before the server takes a request.
// The binary embeds the migrations, so a fresh volume needs nothing beyond
// `docker compose up` (ADR-0033).
func migrate(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("reach database: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	for _, r := range results {
		slog.Info("product: migrated", "version", r.Source.Version, "path", r.Source.Path)
	}
	return nil
}
