package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mookle/fennel/apps/product/internal/catalogue"
	"github.com/mookle/fennel/apps/product/internal/money"
)

// productReader is the part of catalogue.Store that GET /v1/products needs.
type productReader interface {
	Products(ctx context.Context, f catalogue.Filter) (catalogue.Page, error)
}

// The contract's bounds on page_size.
const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// statuses is the contract's ProductStatus enum.
var statuses = map[catalogue.Status]struct{}{
	catalogue.StatusIncomplete: {},
	catalogue.StatusActive:     {},
	catalogue.StatusHidden:     {},
	catalogue.StatusArchived:   {},
	catalogue.StatusReported:   {},
	catalogue.StatusDeleted:    {},
}

// listProducts serves GET /v1/products.
type listProducts struct {
	store productReader
}

func (h listProducts) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, details := parseFilter(r.URL.Query())
	if len(details) > 0 {
		writeError(w, r, http.StatusBadRequest, codeInvalidRequest,
			"the query parameters are invalid", details...)
		return
	}

	page, err := h.store.Products(r.Context(), f)
	if err != nil {
		writeInternal(w, r, err)
		return
	}

	body := productPageJSON{
		Items:    make([]productJSON, 0, len(page.Products)),
		Page:     f.Page,
		PageSize: f.PageSize,
		Total:    page.Total,
	}
	for _, p := range page.Products {
		body.Items = append(body.Items, toProductJSON(p))
	}
	writeJSON(w, r, http.StatusOK, body)
}

// parseFilter reads the query parameters into a Filter, and names every
// parameter it rejects. It fills the contract's defaults, so the Filter
// reaches the store complete and the response can echo the page it asked for.
//
// An empty value counts as absent, which matches what an empty field means
// on Filter. A parameter the contract does not name passes unread.
func parseFilter(params url.Values) (catalogue.Filter, []string) {
	f := catalogue.Filter{
		ShopID:   params.Get("shop_id"),
		Label:    params.Get("label"),
		Query:    params.Get("q"),
		Status:   catalogue.StatusActive,
		Page:     1,
		PageSize: defaultPageSize,
	}
	var details []string

	if s := params.Get("status"); s != "" {
		if _, ok := statuses[catalogue.Status(s)]; ok {
			f.Status = catalogue.Status(s)
		} else {
			details = append(details, fmt.Sprintf("status %q is not a product status", s))
		}
	}

	if s := params.Get("page"); s != "" {
		n, err := strconv.Atoi(s)
		if err == nil && n >= 1 {
			f.Page = n
		} else {
			details = append(details, fmt.Sprintf("page %q is not an integer of 1 or more", s))
		}
	}

	if s := params.Get("page_size"); s != "" {
		n, err := strconv.Atoi(s)
		if err == nil && n >= 1 && n <= maxPageSize {
			f.PageSize = n
		} else {
			details = append(details, fmt.Sprintf("page_size %q is not an integer from 1 to %d", s, maxPageSize))
		}
	}

	return f, details
}

// productPageJSON is the 200 body of GET /v1/products.
type productPageJSON struct {
	Items    []productJSON `json:"items"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Total    int           `json:"total"`
}

// productJSON is the contract's Product schema.
//
// base_price is absent from the object while the product has no SKUs, rather
// than null, because the contract does not make it nullable (ADR-0027,
// ADR-0031).
type productJSON struct {
	ID          uuid.UUID        `json:"id"`
	ShopID      string           `json:"shop_id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Status      catalogue.Status `json:"status"`
	BasePrice   *money.Money     `json:"base_price,omitempty"`
	Currency    money.Currency   `json:"currency"`
	LabelSlugs  []string         `json:"label_slugs"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// toProductJSON maps a product to the wire. The timestamps go out in UTC, so
// the offset a response carries never depends on the server's time zone.
func toProductJSON(p catalogue.Product) productJSON {
	out := productJSON{
		ID:          p.ID,
		ShopID:      p.ShopID,
		Name:        p.Name,
		Description: p.Description,
		Status:      p.Status,
		Currency:    p.Currency,
		LabelSlugs:  p.LabelSlugs,
		CreatedAt:   p.CreatedAt.UTC(),
		UpdatedAt:   p.UpdatedAt.UTC(),
	}
	if p.BasePrice.Valid {
		out.BasePrice = &p.BasePrice.Money
	}
	if out.LabelSlugs == nil {
		out.LabelSlugs = []string{}
	}
	return out
}
