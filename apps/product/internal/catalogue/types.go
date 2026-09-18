// Package catalogue owns the descriptive aspects of a product.
package catalogue

import (
	"time"

	"github.com/google/uuid"
	"github.com/mookle/fennel/apps/product/internal/money"
)

// Product is the list-item projection of a product.
type Product struct {
	ID          uuid.UUID
	ShopID      string
	Name        string
	Description string
	Status      Status
	Currency    money.Currency
	BasePrice   money.NullMoney
	LabelSlugs  []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Status is a product's lifecycle state. The contract closes the set to the
// six values below, and docs/services/product.md defines what each one means
// and which transitions the service allows.
//
// A product's current status is the status of the most recent
// product_status_history entry, not a field of the product row (ADR-0019).
type Status string

const (
	StatusIncomplete Status = "incomplete"
	StatusActive     Status = "active"
	StatusHidden     Status = "hidden"
	StatusArchived   Status = "archived"
	StatusReported   Status = "reported"
	StatusDeleted    Status = "deleted"
)

// ProductDetail is the single product projection.
type ProductDetail struct {
	Product
	Attributes []Attribute
	Skus       []Sku
}

// Attribute encodes a product variation, for example "Size". A product holds
// 0-n of them.
type Attribute struct {
	ID        uuid.UUID
	ProductID uuid.UUID
	Name      string
	Options   []AttributeOption
}

// AttributeOption represents the concrete value an attribute can take. For
// example "Large". An attribute holds 1-n of them.
type AttributeOption struct {
	ID   uuid.UUID
	Name string
}

// Sku is the "tangible" unit of sale.
//
// Code is not a column on the SKU row. It comes from the code allocation in
// sku_codes (ADR-0026).
//
// AvailableQuantity comes from sku_stock, stored in separate table because it
// has a much higher write profile than skus does.
type Sku struct {
	ID                uuid.UUID
	ProductID         uuid.UUID
	Code              string
	Price             money.Money
	AvailableQuantity int
	OptionIDs         []uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ResolvedSku is a denormalised read model for wire transmission.
//
// It is the SKU row joined with its product's name, description and shop_id,
// the resolved option names, and the stock. Denormalising the data means a
// Purchase line can be created without a follow-up read (ADR-0004).
type ResolvedSku struct {
	SkuID             uuid.UUID
	ProductID         uuid.UUID
	ShopID            string
	SkuCode           string
	Name              string
	Description       string
	Price             money.Money
	AvailableQuantity int
	Options           []ResolvedOption
}

// ResolvedOption names one applied option and the attribute it belongs to.
type ResolvedOption struct {
	Attribute string
	Value     string
}

// Label is a platform-wide categorisation tag shared across shops. Aliases stop
// fragmentation, so "Jewellery" and "Jewelry" resolve to one label.
type Label struct {
	ID      uuid.UUID
	Name    string
	Slug    string
	Aliases []string
}
