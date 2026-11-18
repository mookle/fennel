package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mookle/fennel/apps/product/internal/catalogue"
)

// skuIDsQuery builds a batchGet target with each id as one sku_ids value.
func skuIDsQuery(ids ...string) string {
	if len(ids) == 0 {
		return "/v1/skus:batchGet"
	}
	return "/v1/skus:batchGet?sku_ids=" + strings.Join(ids, "&sku_ids=")
}

// randomIDs returns n distinct lowercase hyphenated UUIDs.
func randomIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = uuid.NewString()
	}
	return ids
}

// repeatID returns id n times.
func repeatID(id string, n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = id
	}
	return ids
}

func TestBatchGetSkusWritesTheContractExample(t *testing.T) {
	// Whatever ids arrive, the facade answers with the contract's SkuBatch
	// example (ADR-0015).
	rec := get(t, &fakeProducts{}, skuIDsQuery(uuid.NewString()))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	const example = `{
	  "items": [
	    {
	      "sku_id": "019df244-fa20-72b6-a1cf-300ff7354d1d",
	      "product_id": "019df242-3b00-7b38-a574-35b288b2eef2",
	      "shop_id": "shop_42",
	      "sku_code": "APN0MTS",
	      "name": "A Pretty necklace",
	      "description": "Hand-made silver pendant on a fine chain.",
	      "price": {"amount": "24.0000", "currency": "GBP"},
	      "available_quantity": 3,
	      "options": [{"attribute": "Material", "value": "Silver"}]
	    },
	    {
	      "sku_id": "019dfdbc-4f70-7952-bc65-165ee40433e7",
	      "product_id": "019dfdbc-4f70-7746-b01b-05c48c3c3156",
	      "shop_id": "shop_7",
	      "sku_code": "BWS0",
	      "name": "Beeswax Wrap Set",
	      "price": {"amount": "12.5000", "currency": "GBP"},
	      "available_quantity": 0,
	      "options": []
	    }
	  ],
	  "missing": ["019dfdbc-4f70-75fd-9edf-f8c045786f4a"]
	}`
	var want any
	if err := json.Unmarshal([]byte(example), &want); err != nil {
		t.Fatal(err)
	}
	if got := decode(t, rec); !reflect.DeepEqual(got, want) {
		t.Errorf("body = %s\nwant %s", rec.Body, example)
	}
}

func TestBatchGetSkusStatus(t *testing.T) {
	id := uuid.NewString()
	tests := []struct {
		name string
		ids  []string
		want int
	}{
		{"one id", []string{id}, http.StatusOK},
		{"100 distinct ids", randomIDs(100), http.StatusOK},
		{"100 copies of one id", repeatID(id, 100), http.StatusOK},
		{"101 distinct ids", randomIDs(101), http.StatusBadRequest},
		{"101 copies of one id", repeatID(id, 101), http.StatusBadRequest},
		{"no sku_ids", nil, http.StatusBadRequest},
		{"an empty value", []string{""}, http.StatusBadRequest},
		{"a value that is not a UUID", []string{id, "not-a-uuid"}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, &fakeProducts{}, skuIDsQuery(tt.ids...))
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
			if tt.want != http.StatusBadRequest {
				return
			}
			body, ok := decode(t, rec).(map[string]any)
			if !ok || body["code"] != codeInvalidRequest {
				t.Errorf("body = %s, want an Error with code %q", rec.Body, codeInvalidRequest)
			}
		})
	}
}

func TestBatchGetSkusRejectsPost(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, skuIDsQuery(uuid.NewString()), nil)
	New(&fakeProducts{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestParseSkuIDsNamesEveryBadValue(t *testing.T) {
	ids, details := parseSkuIDs(map[string][]string{"sku_ids": {"a", uuid.NewString(), "b"}})
	if ids != nil {
		t.Errorf("ids = %v, want nil", ids)
	}
	want := []string{`sku_ids "a" is not a UUID`, `sku_ids "b" is not a UUID`}
	if !reflect.DeepEqual(details, want) {
		t.Errorf("details = %q, want %q", details, want)
	}
}

func TestParseSkuIDsKeepsOrderAndDuplicates(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	ids, details := parseSkuIDs(map[string][]string{"sku_ids": {a.String(), b.String(), a.String()}})
	if details != nil {
		t.Fatalf("details = %q, want none", details)
	}
	if want := []uuid.UUID{a, b, a}; !reflect.DeepEqual(ids, want) {
		t.Errorf("ids = %v, want %v", ids, want)
	}
}

func TestToResolvedSkuJSONLeavesOutAnEmptyDescription(t *testing.T) {
	out := toResolvedSkuJSON(catalogue.ResolvedSku{Price: mustMoney(t, "1", "GBP")})
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["description"]; ok {
		t.Errorf("description present in %s, want absent", b)
	}
	if opts, ok := got["options"].([]any); !ok || len(opts) != 0 {
		t.Errorf("options = %v, want an empty array", got["options"])
	}
}
