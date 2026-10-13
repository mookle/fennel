package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mookle/fennel/apps/product/internal/catalogue"
	"github.com/mookle/fennel/apps/product/internal/money"
)

// fakeProducts records the Filter it receives and answers with a fixed page.
type fakeProducts struct {
	page   catalogue.Page
	err    error
	called bool
	got    catalogue.Filter
}

func (f *fakeProducts) Products(_ context.Context, filter catalogue.Filter) (catalogue.Page, error) {
	f.called = true
	f.got = filter
	return f.page, f.err
}

// get serves one request through the routes New builds.
func get(t *testing.T, store productReader, target string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	New(store).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

// decode reads a body into generic JSON, so a test compares what went over
// the wire rather than what a Go type reads back.
func decode(t *testing.T, rec *httptest.ResponseRecorder) any {
	t.Helper()

	var v any
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
	return v
}

func mustMoney(t *testing.T, amount, currency string) money.Money {
	t.Helper()

	a, err := money.ParseAmount(amount)
	if err != nil {
		t.Fatal(err)
	}
	return money.Money{Amount: a, Currency: mustCurrency(t, currency)}
}

func mustCurrency(t *testing.T, code string) money.Currency {
	t.Helper()

	c, err := money.ParseCurrency(code)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()

	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func TestListProductsWritesTheContractExample(t *testing.T) {
	// The store answers with the products of the contract's ProductPage
	// example, so the body must equal that example (ADR-0015). The second
	// product's timestamps arrive in another zone, and go out in UTC.
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeProducts{page: catalogue.Page{
		Total: 2,
		Products: []catalogue.Product{
			{
				ID:          uuid.MustParse("019df242-3b00-7b38-a574-35b288b2eef2"),
				ShopID:      "shop_42",
				Name:        "A Pretty necklace",
				Description: "Hand-made silver pendant on a fine chain.",
				Status:      catalogue.StatusActive,
				Currency:    mustCurrency(t, "GBP"),
				BasePrice:   money.NullMoney{Money: mustMoney(t, "24", "GBP"), Valid: true},
				LabelSlugs:  []string{"jewellery", "silver"},
				CreatedAt:   mustTime(t, "2026-05-04T09:12:00Z"),
				UpdatedAt:   mustTime(t, "2026-05-04T09:12:00Z"),
			},
			{
				ID:          uuid.MustParse("019dfdbc-4f70-7746-b01b-05c48c3c3156"),
				ShopID:      "shop_7",
				Name:        "Beeswax Wrap Set",
				Description: "Three reusable food wraps.",
				Status:      catalogue.StatusActive,
				Currency:    mustCurrency(t, "GBP"),
				BasePrice:   money.NullMoney{Money: mustMoney(t, "12.5", "GBP"), Valid: true},
				LabelSlugs:  []string{"kitchen"},
				CreatedAt:   mustTime(t, "2026-05-06T14:41:10Z").In(london),
				UpdatedAt:   mustTime(t, "2026-06-01T08:00:00Z").In(london),
			},
		},
	}}

	rec := get(t, store, "/v1/products")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	const example = `{
	  "items": [
	    {
	      "id": "019df242-3b00-7b38-a574-35b288b2eef2",
	      "shop_id": "shop_42",
	      "name": "A Pretty necklace",
	      "description": "Hand-made silver pendant on a fine chain.",
	      "status": "active",
	      "base_price": {"amount": "24.0000", "currency": "GBP"},
	      "currency": "GBP",
	      "label_slugs": ["jewellery", "silver"],
	      "created_at": "2026-05-04T09:12:00Z",
	      "updated_at": "2026-05-04T09:12:00Z"
	    },
	    {
	      "id": "019dfdbc-4f70-7746-b01b-05c48c3c3156",
	      "shop_id": "shop_7",
	      "name": "Beeswax Wrap Set",
	      "description": "Three reusable food wraps.",
	      "status": "active",
	      "base_price": {"amount": "12.5000", "currency": "GBP"},
	      "currency": "GBP",
	      "label_slugs": ["kitchen"],
	      "created_at": "2026-05-06T14:41:10Z",
	      "updated_at": "2026-06-01T08:00:00Z"
	    }
	  ],
	  "page": 1,
	  "page_size": 20,
	  "total": 2
	}`
	var want any
	if err := json.Unmarshal([]byte(example), &want); err != nil {
		t.Fatal(err)
	}
	if got := decode(t, rec); !reflect.DeepEqual(got, want) {
		t.Errorf("body = %s\nwant %s", rec.Body, example)
	}
}

// listOne serves a page that holds p alone, and returns its item as generic
// JSON.
func listOne(t *testing.T, p catalogue.Product) map[string]any {
	t.Helper()

	store := &fakeProducts{page: catalogue.Page{Total: 1, Products: []catalogue.Product{p}}}
	rec := get(t, store, "/v1/products")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	return decode(t, rec).(map[string]any)["items"].([]any)[0].(map[string]any)
}

func TestListProductsOmitsAnAbsentBasePrice(t *testing.T) {
	// A product with no SKUs has no base_price at all. The contract does not
	// make the field nullable, so it leaves the object rather than going out
	// as null (ADR-0027, ADR-0031).
	item := listOne(t, catalogue.Product{
		ID:       uuid.MustParse("0194f000-0005-7000-8000-000000000005"),
		Status:   catalogue.StatusIncomplete,
		Currency: mustCurrency(t, "GBP"),
	})
	if v, ok := item["base_price"]; ok {
		t.Errorf("base_price = %v, want absent", v)
	}
}

func TestListProductsWritesNoLabelsAsAnEmptyList(t *testing.T) {
	// The store reads a product with no labels as nil slugs, and the wire
	// carries that as [] rather than null.
	item := listOne(t, catalogue.Product{
		ID:       uuid.MustParse("0194f000-0006-7000-8000-000000000006"),
		Status:   catalogue.StatusActive,
		Currency: mustCurrency(t, "GBP"),
	})
	if v, ok := item["label_slugs"].([]any); !ok || len(v) != 0 {
		t.Errorf("label_slugs = %v, want []", item["label_slugs"])
	}
}

func TestListProductsWritesAnEmptyPageAsAList(t *testing.T) {
	store := &fakeProducts{page: catalogue.Page{Total: 4}}

	rec := get(t, store, "/v1/products?page=9")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}

	body := decode(t, rec).(map[string]any)
	if v, ok := body["items"].([]any); !ok || len(v) != 0 {
		t.Errorf("items = %v, want []", body["items"])
	}
	if body["total"] != 4.0 {
		t.Errorf("total = %v, want 4", body["total"])
	}
}

func TestListProductsParsesTheQuery(t *testing.T) {
	cases := []struct {
		name   string
		target string
		want   catalogue.Filter
	}{
		{
			name:   "no parameters ask for the first page of the active catalogue",
			target: "/v1/products",
			want:   catalogue.Filter{Status: catalogue.StatusActive, Page: 1, PageSize: 20},
		},
		{
			name:   "every parameter reaches the filter",
			target: "/v1/products?shop_id=shop_42&label=silver&q=necklace&status=hidden&page=3&page_size=100",
			want: catalogue.Filter{
				ShopID: "shop_42", Label: "silver", Query: "necklace",
				Status: catalogue.StatusHidden, Page: 3, PageSize: 100,
			},
		},
		{
			name:   "an empty value counts as absent",
			target: "/v1/products?shop_id=&label=&q=&status=&page=&page_size=",
			want:   catalogue.Filter{Status: catalogue.StatusActive, Page: 1, PageSize: 20},
		},
		{
			// The store escapes the LIKE wildcards, so the handler passes
			// the query through as the caller wrote it.
			name:   "q arrives decoded and unescaped",
			target: "/v1/products?q=50%25+off_now",
			want: catalogue.Filter{
				Query: "50% off_now", Status: catalogue.StatusActive, Page: 1, PageSize: 20,
			},
		},
		{
			name:   "unknown parameters pass unread",
			target: "/v1/products?sort=name",
			want:   catalogue.Filter{Status: catalogue.StatusActive, Page: 1, PageSize: 20},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := &fakeProducts{}
			rec := get(t, store, c.target)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
			}
			if store.got != c.want {
				t.Errorf("filter = %+v, want %+v", store.got, c.want)
			}

			// The response echoes the page the filter asked for.
			body := decode(t, rec).(map[string]any)
			if body["page"] != float64(c.want.Page) || body["page_size"] != float64(c.want.PageSize) {
				t.Errorf("page, page_size = %v, %v, want %d, %d",
					body["page"], body["page_size"], c.want.Page, c.want.PageSize)
			}
		})
	}
}

func TestListProductsRejectsAnInvalidQuery(t *testing.T) {
	cases := []struct {
		name    string
		target  string
		details int
	}{
		{"an unknown status", "/v1/products?status=sold", 1},
		{"a page of zero", "/v1/products?page=0", 1},
		{"a page that is not an integer", "/v1/products?page=two", 1},
		{"a page size of zero", "/v1/products?page_size=0", 1},
		{"a page size above the maximum", "/v1/products?page_size=101", 1},
		{"every bad parameter at once", "/v1/products?status=sold&page=-1&page_size=500", 3},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := &fakeProducts{}
			rec := get(t, store, c.target)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body)
			}
			if store.called {
				t.Error("the store was read, want the request rejected first")
			}

			var body errorJSON
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode %s: %v", rec.Body, err)
			}
			if body.Code != codeInvalidRequest {
				t.Errorf("code = %q, want %q", body.Code, codeInvalidRequest)
			}
			if len(body.Details) != c.details {
				t.Errorf("details = %q, want %d of them", body.Details, c.details)
			}
		})
	}
}

func TestListProductsHidesAStoreError(t *testing.T) {
	store := &fakeProducts{err: errors.New("catalogue: query products: connection refused")}

	rec := get(t, store, "/v1/products")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", rec.Code, rec.Body)
	}

	var body errorJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
	if body.Code != codeInternal {
		t.Errorf("code = %q, want %q", body.Code, codeInternal)
	}
	if body.Message != "internal error" {
		t.Errorf("message = %q, want the cause withheld", body.Message)
	}
}

func TestListProductsAnswersGetOnly(t *testing.T) {
	rec := httptest.NewRecorder()
	New(&fakeProducts{}).ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/v1/products", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE status = %d, want 405", rec.Code)
	}
}
