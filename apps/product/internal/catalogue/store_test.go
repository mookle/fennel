package catalogue

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// dsnEnv names the environment variable that points these tests at a
// Postgres server. Compose runs one (ADR-0033). The tests skip when it is
// unset, so `go test ./...` passes without a database.
const dsnEnv = "FENNEL_PRODUCT_DSN"

// The fixture these tests read. testdata/catalogue.sql fixes every id, so a
// test can name a product rather than infer it from a position.
const (
	copperRing    = "0194f000-0001-7000-8000-000000000001"
	linenApron    = "0194f000-0002-7000-8000-000000000002"
	oakSpoon      = "0194f000-0003-7000-8000-000000000003"
	beeswaxCandle = "0194f000-0004-7000-8000-000000000004"
	silverCuff    = "0194f000-0005-7000-8000-000000000005"
	slateCoaster  = "0194f000-0006-7000-8000-000000000006"
)

// fixtureDB reads the fixture schema. TestMain opens it, and it stays nil
// when no DSN is set.
var fixtureDB *sql.DB

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	dsn := os.Getenv(dsnEnv)
	if dsn == "" {
		return m.Run()
	}

	db, drop, err := openFixture(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "catalogue: %v\n", err)
		return 1
	}
	defer drop()

	fixtureDB = db
	return m.Run()
}

// openFixture creates a schema of its own for this run, migrates it and
// loads the fixture. Every connection in the returned pool resolves names in
// that schema first, so the tests never see another run's rows or the
// database's own. drop closes the pool and removes the schema.
func openFixture(ctx context.Context, dsn string) (db *sql.DB, drop func(), err error) {
	// Ensure the dsn is good before executing CREATE SCHEMA
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", dsnEnv, err)
	}

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", dsnEnv, err)
	}

	schema := "catalogue_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		return nil, nil, fmt.Errorf("create schema: %w", err)
	}

	cfg.RuntimeParams["search_path"] = schema
	db = stdlib.OpenDB(*cfg)

	drop = func() {
		db.Close()
		if _, err := admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			fmt.Fprintf(os.Stderr, "catalogue: drop schema %s: %v\n", schema, err)
		}
		admin.Close()
	}
	defer func() {
		if err != nil {
			drop()
		}
	}()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../migrations"))
	if err != nil {
		return nil, nil, fmt.Errorf("load migrations: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return nil, nil, fmt.Errorf("migrate: %w", err)
	}

	fixture, err := os.ReadFile("testdata/catalogue.sql")
	if err != nil {
		return nil, nil, fmt.Errorf("read fixture: %w", err)
	}
	if _, err := db.ExecContext(ctx, string(fixture)); err != nil {
		return nil, nil, fmt.Errorf("load fixture: %w", err)
	}
	return db, drop, nil
}

// openStore reads the fixture, or skips the test when no DSN is set. The
// tests only read, so they share one schema and need no reset.
func openStore(t *testing.T) *Store {
	t.Helper()

	if fixtureDB == nil {
		t.Skipf("%s is unset, so there is no database to read", dsnEnv)
	}
	return NewStore(fixtureDB)
}

// ids lists the products of a page in the order they arrived.
func ids(products []Product) []string {
	out := make([]string, len(products))
	for i, p := range products {
		out[i] = p.ID.String()
	}
	return out
}

func TestProductsFilters(t *testing.T) {
	store := openStore(t)

	cases := []struct {
		name  string
		f     Filter
		want  []string
		total int
	}{
		{
			// The zero value is the public read: active products, newest
			// first, across every shop. The beeswax candle was active
			// before it went hidden, so its absence shows the most recent
			// history row wins.
			name:  "the zero filter reads the active catalogue",
			f:     Filter{},
			want:  []string{slateCoaster, oakSpoon, linenApron, copperRing},
			total: 4,
		},
		{
			// The beeswax candle is hidden, and the silver cuff is
			// incomplete. Neither appears above.
			name:  "hidden reads back on its own",
			f:     Filter{Status: StatusHidden},
			want:  []string{beeswaxCandle},
			total: 1,
		},
		{
			name:  "shop_id narrows the page",
			f:     Filter{ShopID: "shop-thistle"},
			want:  []string{linenApron, copperRing},
			total: 2,
		},
		{
			name:  "label narrows the page to the canonical slug",
			f:     Filter{Label: "homeware"},
			want:  []string{oakSpoon, linenApron},
			total: 2,
		},
		{
			name:  "shop_id and label narrow together",
			f:     Filter{ShopID: "shop-bramble", Label: "homeware"},
			want:  []string{oakSpoon},
			total: 1,
		},
		{
			name:  "a shop with no products reads empty",
			f:     Filter{ShopID: "shop-nettle"},
			want:  []string{},
			total: 0,
		},
		{
			// An alias resolves at assignment, so the filter never sees one.
			name:  "an alias matches nothing",
			f:     Filter{Label: "jewelry"},
			want:  []string{},
			total: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			products, total, err := store.Products(t.Context(), c.f)
			if err != nil {
				t.Fatalf("Products: %v", err)
			}
			if got := ids(products); !slices.Equal(got, c.want) {
				t.Errorf("Products gave %v, want %v", got, c.want)
			}
			if total != c.total {
				t.Errorf("total = %d, want %d", total, c.total)
			}

			// Every product on the page reports the status the filter
			// asked for, and an empty filter status asks for active.
			want := cmp.Or(c.f.Status, StatusActive)
			for _, p := range products {
				if p.Status != want {
					t.Errorf("%s came back as %q, want %q", p.Name, p.Status, want)
				}
			}
		})
	}
}

func TestProductsReadsTheCurrentStatusWhenTwoRowsShareATimestamp(t *testing.T) {
	// The linen apron takes its incomplete and its active row at one
	// timestamp, so only the id tells the two apart.
	store := openStore(t)

	p := find(t, store, Filter{ShopID: "shop-thistle"}, linenApron)
	if p.Status != StatusActive {
		t.Errorf("status = %q, want active", p.Status)
	}
}

func TestProductsReadsAnAbsentBasePrice(t *testing.T) {
	// The silver cuff has no SKUs, so it has no base_price at all, and that
	// is absence of the whole value rather than a zero amount (ADR-0031).
	store := openStore(t)

	p := find(t, store, Filter{Status: StatusIncomplete}, silverCuff)
	if p.BasePrice.Valid {
		t.Errorf("base_price = %v, want absent", p.BasePrice.Money)
	}
	if p.Currency.String() != "GBP" {
		t.Errorf("currency = %q, want GBP", p.Currency)
	}
}

func TestProductsReadsABasePriceInTheProductsCurrency(t *testing.T) {
	// A present amount takes the currency from the product row, which is
	// the shop's stand-in (ADR-0028, ADR-0029, ADR-0032).
	store := openStore(t)

	p := find(t, store, Filter{}, oakSpoon)
	if !p.BasePrice.Valid {
		t.Fatal("base_price came back absent")
	}
	if got := p.BasePrice.Money.String(); got != "9.9500 EUR" {
		t.Errorf("base_price = %q, want \"9.9500 EUR\"", got)
	}
	if p.Currency.String() != "EUR" {
		t.Errorf("currency = %q, want EUR", p.Currency)
	}
}

func TestProductsReadsLabelSlugs(t *testing.T) {
	store := openStore(t)

	// The oak spoon carries two labels, and the aggregate sorts them.
	spoon := find(t, store, Filter{}, oakSpoon)
	if want := []string{"homeware", "woodwork"}; !slices.Equal(spoon.LabelSlugs, want) {
		t.Errorf("label_slugs = %v, want %v", spoon.LabelSlugs, want)
	}

	// The slate coaster carries none, which is an empty list rather than an
	// absent one.
	coaster := find(t, store, Filter{}, slateCoaster)
	if coaster.LabelSlugs == nil {
		t.Error("label_slugs came back nil, want an empty slice")
	}
	if len(coaster.LabelSlugs) != 0 {
		t.Errorf("label_slugs = %v, want empty", coaster.LabelSlugs)
	}
}

func TestProductsReadsTheTimestamps(t *testing.T) {
	// updated_at equals created_at on an untouched row, and neither is null
	// (ADR-0018).
	store := openStore(t)

	p := find(t, store, Filter{}, copperRing)
	if p.CreatedAt.IsZero() {
		t.Error("created_at came back zero")
	}
	if !p.UpdatedAt.Equal(p.CreatedAt) {
		t.Errorf("updated_at = %v, want created_at %v", p.UpdatedAt, p.CreatedAt)
	}
}

func TestProductsPaginates(t *testing.T) {
	store := openStore(t)

	// The total reports the four products the filter matched, not the two
	// on the page.
	first, total, err := store.Products(t.Context(), Filter{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("Products(page 1): %v", err)
	}
	if want := []string{slateCoaster, oakSpoon}; !slices.Equal(ids(first), want) {
		t.Errorf("page 1 of 2 gave %v, want %v", ids(first), want)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}

	// The second page of three holds the one product the first page left.
	second, total, err := store.Products(t.Context(), Filter{Page: 2, PageSize: 3})
	if err != nil {
		t.Fatalf("Products(page 2): %v", err)
	}
	if want := []string{copperRing}; !slices.Equal(ids(second), want) {
		t.Errorf("page 2 of 3 gave %v, want %v", ids(second), want)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}

	// A page past the end is empty, and the total still counts the matches,
	// which the window count alone cannot give.
	past, total, err := store.Products(t.Context(), Filter{Page: 3, PageSize: 3})
	if err != nil {
		t.Fatalf("Products(page 3): %v", err)
	}
	if len(past) != 0 {
		t.Errorf("page 3 of 3 gave %v, want empty", ids(past))
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
}

// find reads a page and returns the one product it expected to see.
func find(t *testing.T, store *Store, f Filter, id string) Product {
	t.Helper()

	products, _, err := store.Products(t.Context(), f)
	if err != nil {
		t.Fatalf("Products: %v", err)
	}
	for _, p := range products {
		if p.ID.String() == id {
			return p
		}
	}
	t.Fatalf("%s is not on the page %v", id, ids(products))
	return Product{}
}
