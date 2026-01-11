package domain

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// --- Interfaces ---

type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Category struct {
	ID         string     `json:"id" gorm:"primaryKey"`
	Name       string     `json:"name"`
	Slug       string     `json:"slug" gorm:"uniqueIndex"`
	ParentID   *string    `json:"parentId"`
	Parent     *Category  `json:"-" gorm:"foreignKey:ParentID"`
	Children   []Category `json:"children" gorm:"foreignKey:ParentID"`
	OrderIndex int        `json:"orderIndex" gorm:"default:0"`
	Icon       string     `json:"icon"`
	IsActive   bool       `json:"isActive" gorm:"default:true"`
}

type Product struct {
	ID                string    `json:"id" gorm:"primaryKey"`
	Name              string    `json:"name"`
	Slug              string    `json:"slug" gorm:"uniqueIndex"`
	SKU               string    `json:"sku" gorm:"uniqueIndex"` // Robust Inventory
	Description       string    `json:"description"`
	BasePrice         float64   `json:"basePrice"`
	SalePrice         *float64  `json:"salePrice"`
	Stock             int       `json:"stock" gorm:"default:0"` // Main inventory
	StockStatus       string    `json:"stockStatus"`
	LowStockThreshold int       `json:"lowStockThreshold" gorm:"default:5"`
	IsFeatured        bool      `json:"isFeatured"`
	IsActive          bool      `json:"isActive" gorm:"default:true"`
	CategoryID        string    `json:"categoryId"`
	Category          Category  `json:"category" gorm:"foreignKey:CategoryID"`
	Media             JSONB     `json:"media" gorm:"type:jsonb"`
	Attributes        JSONB     `json:"attributes" gorm:"type:jsonb"`
	Specs             JSONB     `json:"specifications" gorm:"type:jsonb"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	Variants          []Variant `json:"variants" gorm:"foreignKey:ProductID"`
}

type Variant struct {
	ID        string `json:"id" gorm:"primaryKey"`
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Stock     int    `json:"stock"`
	SKU       string `json:"sku"` // Optional: Variant specific SKU
}

type InventoryLog struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	ProductID    string    `json:"productId"`
	VariantID    *string   `json:"variantId"`
	ChangeAmount int       `json:"changeAmount"` // +10 or -5
	Reason       string    `json:"reason"`       // order_placed, restock, return, adjustment, cancelled
	ReferenceID  string    `json:"referenceId"`  // OrderID or Admin UserID
	CreatedAt    time.Time `json:"createdAt"`
}

// --- Interfaces ---

type ProductRepository interface {
	GetCategoryTree(ctx context.Context) ([]Category, error)
	GetProducts(ctx context.Context, filter ProductFilter) ([]Product, int64, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	GetProductByID(ctx context.Context, id string) (*Product, error)
	UpdateStock(ctx context.Context, productID string, quantity int, reason, referenceID string) error

	// Admin Management
	CreateProduct(ctx context.Context, product *Product) error
	UpdateProduct(ctx context.Context, product *Product) error
	DeleteProduct(ctx context.Context, id string) error
}

type ProductFilter struct {
	CategorySlug string
	Query        string
	MinPrice     float64
	MaxPrice     float64
	Sort         string // newest, price_asc, price_desc
	Limit        int
	Offset       int
}

// --- Custom Types ---

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &j)
}
