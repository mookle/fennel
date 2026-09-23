package catalogue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Store reads the catalogue from Postgres.
type Store struct {
	db *sql.DB
}

// NewStore wraps a pool. The caller owns the pool and closes it.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Filter narrows a catalogue query. The zero value asks for the first page of
// active products across every shop, which is the public read.
type Filter struct {
	// ShopID limits the page to one shop. An empty value matches every shop.
	ShopID string

	// Label is a canonical label slug. The contract takes the canonical
	// slug, so an alias resolves at assignment rather than here. An empty
	// value matches every label, and a product with no labels at all.
	Label string

	// Status is the status the product currently holds. An empty value
	// means active, because the public catalogue query returns active
	// products only, so no zero-value filter can leak a hidden product.
	Status Status

	// Page counts from one, and PageSize takes the contract's default of 20
	// and its maximum of 100. The handler validates a request before it
	// reaches here; these bounds keep a zero value from reaching LIMIT.
	Page     int
	PageSize int
}

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// normalise fills the defaults and clamps the page bounds.
func (f Filter) normalise() Filter {
	if f.Status == "" {
		f.Status = StatusActive
	}
	if f.Page < 1 {
		f.Page = 1
	}
	switch {
	case f.PageSize < 1:
		f.PageSize = defaultPageSize
	case f.PageSize > maxPageSize:
		f.PageSize = maxPageSize
	}
	return f
}

// productsMatch is the FROM and WHERE both catalogue queries share, so the
// page and its fallback count can never filter differently.
//
// The status comes from the most recent product_status_history row, which a
// LATERAL join resolves per product (ADR-0019). The id breaks a tie on
// created_at, so two transitions written at one timestamp still name one
// current status.
const productsMatch = `
  FROM products p
  JOIN LATERAL (SELECT h.status FROM product_status_history h
                 WHERE h.product_id = p.id
                 ORDER BY h.created_at DESC, h.id DESC LIMIT 1) s ON TRUE
 WHERE s.status = $1
   AND ($2::text IS NULL OR p.shop_id = $2)
   AND ($3::text IS NULL OR EXISTS (
         SELECT 1 FROM product_labels pl JOIN labels l ON l.id = pl.label_id
          WHERE pl.product_id = p.id AND l.slug = $3))`

// productsQuery reads one page of the catalogue and its total in one pass.
//
// base_price routes the absent case to a plain NULL rather than to a
// composite with an empty amount, which is what NullMoney scans (ADR-0031,
// ADR-0032). A present amount takes the product's currency (ADR-0028).
//
// The label slugs come back as JSON text rather than as a text[], because a
// Postgres array needs its own parser and JSON does not.
//
// COUNT(*) OVER () counts the rows the filter matched, before LIMIT cuts the
// page. Taking the total in this pass rather than in a second query removes
// any chance the two disagree. A window count rides on the rows, though, so
// a page past the end carries none, and productsCountQuery covers that case.
//
// The order is fixed here because the contract fixes none, and offset
// pagination needs a total order to cut a stable page. created_at is the
// recency column (ADR-0018), and the id breaks a tie.
const productsQuery = `
SELECT p.id, p.shop_id, p.name, p.description,
       s.status, p.currency,
       CASE WHEN p.base_price IS NULL THEN NULL
            ELSE (p.base_price, p.currency)::money_value END,
       (SELECT COALESCE(json_agg(l.slug ORDER BY l.slug), '[]')
          FROM product_labels pl JOIN labels l ON l.id = pl.label_id
         WHERE pl.product_id = p.id),
       p.created_at, p.updated_at,
       COUNT(*) OVER () AS total` + productsMatch + `
 ORDER BY p.created_at DESC, p.id DESC
 LIMIT $4 OFFSET $5`

// productsCountQuery counts the matches on its own. Products runs it only
// when a page past the first comes back empty.
const productsCountQuery = `SELECT COUNT(*)` + productsMatch

// Products returns one page of the catalogue and the number of products the
// filter matched, which is the contract's total and not the length of the
// page. An empty page is not an error, and a page past the end still
// reports the total.
func (s *Store) Products(ctx context.Context, f Filter) ([]Product, int, error) {
	f = f.normalise()

	status, shopID, label := string(f.Status), nullIfEmpty(f.ShopID), nullIfEmpty(f.Label)
	offset := (f.Page - 1) * f.PageSize

	rows, err := s.db.QueryContext(ctx, productsQuery,
		status, shopID, label, f.PageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("catalogue: query products: %w", err)
	}
	defer rows.Close()

	// An empty slice rather than a nil one, so an empty page encodes as []
	// and not as null.
	products := []Product{}
	total := 0
	for rows.Next() {
		var (
			p          Product
			labelSlugs []byte
		)
		if err := rows.Scan(
			&p.ID,
			&p.ShopID,
			&p.Name,
			&p.Description,
			&p.Status,
			&p.Currency,
			&p.BasePrice,
			&labelSlugs,
			&p.CreatedAt,
			&p.UpdatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("catalogue: scan product: %w", err)
		}
		if err := json.Unmarshal(labelSlugs, &p.LabelSlugs); err != nil {
			return nil, 0, fmt.Errorf("catalogue: read label slugs of %s: %w", p.ID, err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("catalogue: read products: %w", err)
	}

	// An empty first page means nothing matched, so the zero total is right.
	// An empty later page may sit past the end of a non-empty match.
	if len(products) == 0 && offset > 0 {
		if err := s.db.QueryRowContext(ctx, productsCountQuery,
			status, shopID, label).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("catalogue: count products: %w", err)
		}
	}
	return products, total, nil
}

// nullIfEmpty turns an unset filter field into the SQL NULL the query tests
// for, so one statement serves every combination of filters.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
