package httpapi

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/mookle/fennel/apps/product/internal/catalogue"
	"github.com/mookle/fennel/apps/product/internal/money"
)

// maxBatchSkuIDs is the contract's bound on sku_ids. It counts the values
// sent, so a repeated id takes a slot each time it appears.
const maxBatchSkuIDs = 100

// batchGetSkus serves GET /v1/skus:batchGet as a facade. It validates the ids
// and then answers with a fixed batch, whatever ids it was given.
type batchGetSkus struct {
	items   []catalogue.ResolvedSku
	missing []uuid.UUID
}

// newBatchGetSkus returns the facade with the batch of the contract's
// SkuBatch example. The service cannot read the contract at run time, so the
// values are copied here.
func newBatchGetSkus() batchGetSkus {
	return batchGetSkus{
		items: []catalogue.ResolvedSku{
			{
				SkuID:             uuid.MustParse("019df244-fa20-72b6-a1cf-300ff7354d1d"),
				ProductID:         uuid.MustParse("019df242-3b00-7b38-a574-35b288b2eef2"),
				ShopID:            "shop_42",
				SkuCode:           "APN0MTS",
				Name:              "A Pretty necklace",
				Description:       "Hand-made silver pendant on a fine chain.",
				Price:             fixedMoney("24", "GBP"),
				AvailableQuantity: 3,
				Options:           []catalogue.ResolvedOption{{Attribute: "Material", Value: "Silver"}},
			},
			{
				SkuID:             uuid.MustParse("019dfdbc-4f70-7952-bc65-165ee40433e7"),
				ProductID:         uuid.MustParse("019dfdbc-4f70-7746-b01b-05c48c3c3156"),
				ShopID:            "shop_7",
				SkuCode:           "BWS0",
				Name:              "Beeswax Wrap Set",
				Price:             fixedMoney("12.5", "GBP"),
				AvailableQuantity: 0,
			},
		},
		missing: []uuid.UUID{uuid.MustParse("019dfdbc-4f70-75fd-9edf-f8c045786f4a")},
	}
}

// fixedMoney builds a Money from literals, and panics if they do not parse.
func fixedMoney(amount, currency string) money.Money {
	a, err := money.ParseAmount(amount)
	if err != nil {
		panic(err)
	}
	c, err := money.ParseCurrency(currency)
	if err != nil {
		panic(err)
	}
	return money.Money{Amount: a, Currency: c}
}

func (h batchGetSkus) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, details := parseSkuIDs(r.URL.Query()); len(details) > 0 {
		writeError(w, r, http.StatusBadRequest, codeInvalidRequest,
			"the query parameters are invalid", details...)
		return
	}

	body := skuBatchJSON{
		Items:   make([]resolvedSkuJSON, 0, len(h.items)),
		Missing: h.missing,
	}
	for _, s := range h.items {
		body.Items = append(body.Items, toResolvedSkuJSON(s))
	}
	writeJSON(w, r, http.StatusOK, body)
}

// parseSkuIDs reads the repeated sku_ids parameter, and names every reason it
// rejects it: no values, more than maxBatchSkuIDs values, or a value that is
// not a UUID.
func parseSkuIDs(params url.Values) ([]uuid.UUID, []string) {
	values := params["sku_ids"]
	if len(values) == 0 {
		return nil, []string{"sku_ids is required"}
	}
	if len(values) > maxBatchSkuIDs {
		return nil, []string{fmt.Sprintf("sku_ids has %d values, more than %d", len(values), maxBatchSkuIDs)}
	}

	ids := make([]uuid.UUID, 0, len(values))
	var details []string
	for _, v := range values {
		id, err := uuid.Parse(v)
		if err != nil {
			details = append(details, fmt.Sprintf("sku_ids %q is not a UUID", v))
			continue
		}
		ids = append(ids, id)
	}
	if len(details) > 0 {
		return nil, details
	}
	return ids, nil
}

// skuBatchJSON is the 200 body of GET /v1/skus:batchGet.
type skuBatchJSON struct {
	Items   []resolvedSkuJSON `json:"items"`
	Missing []uuid.UUID       `json:"missing"`
}

// resolvedSkuJSON is the contract's ResolvedSku schema. description is
// optional in the contract, so an SKU whose product has none leaves it out.
type resolvedSkuJSON struct {
	SkuID             uuid.UUID            `json:"sku_id"`
	ProductID         uuid.UUID            `json:"product_id"`
	ShopID            string               `json:"shop_id"`
	SkuCode           string               `json:"sku_code"`
	Name              string               `json:"name"`
	Description       string               `json:"description,omitempty"`
	Price             money.Money          `json:"price"`
	AvailableQuantity int                  `json:"available_quantity"`
	Options           []resolvedOptionJSON `json:"options"`
}

// resolvedOptionJSON is one entry of ResolvedSku's options.
type resolvedOptionJSON struct {
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
}

// toResolvedSkuJSON maps a resolved SKU to the wire. An SKU with no options
// carries an empty array, not null.
func toResolvedSkuJSON(s catalogue.ResolvedSku) resolvedSkuJSON {
	out := resolvedSkuJSON{
		SkuID:             s.SkuID,
		ProductID:         s.ProductID,
		ShopID:            s.ShopID,
		SkuCode:           s.SkuCode,
		Name:              s.Name,
		Description:       s.Description,
		Price:             s.Price,
		AvailableQuantity: s.AvailableQuantity,
		Options:           make([]resolvedOptionJSON, 0, len(s.Options)),
	}
	for _, o := range s.Options {
		out.Options = append(out.Options, resolvedOptionJSON{Attribute: o.Attribute, Value: o.Value})
	}
	return out
}
