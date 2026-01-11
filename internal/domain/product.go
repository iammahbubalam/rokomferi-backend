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
	ID       string     `json:"id" gorm:"primaryKey"` // cat_women
	Name     string     `json:"name"`
	Slug     string     `json:"slug" gorm:"uniqueIndex"`
	ParentID *string    `json:"parentId"`
	Parent   *Category  `json:"-" gorm:"foreignKey:ParentID"`
	Children []Category `json:"children" gorm:"foreignKey:ParentID"`
}

type Product struct {
	ID          string    `json:"id" gorm:"primaryKey"` // p_001
	Name        string    `json:"name"`
	Slug        string    `json:"slug" gorm:"uniqueIndex"`
	Description string    `json:"description"`
	BasePrice   float64   `json:"basePrice"`
	SalePrice   *float64  `json:"salePrice"`
	StockStatus string    `json:"stockStatus"` // in_stock, out_of_stock
	IsFeatured  bool      `json:"isFeatured"`
	CategoryID  string    `json:"categoryId"`
	Category    Category  `json:"category" gorm:"foreignKey:CategoryID"`
	Media       JSONB     `json:"media" gorm:"type:jsonb"`
	Attributes  JSONB     `json:"attributes" gorm:"type:jsonb"`
	Specs       JSONB     `json:"specifications" gorm:"type:jsonb"` // material, weave, etc.
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Variants    []Variant `json:"variants" gorm:"foreignKey:ProductID"`
}

type Variant struct {
	ID        string `json:"id" gorm:"primaryKey"`
	ProductID string `json:"productId"`
	Name      string `json:"name"` // Red, XL, etc.
	Stock     int    `json:"stock"`
}

// --- Interfaces ---

type ProductRepository interface {
	GetCategoryTree(ctx context.Context) ([]Category, error)
	GetProducts(ctx context.Context, filter ProductFilter) ([]Product, int64, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	GetProductByID(ctx context.Context, id string) (*Product, error)
	// Create/Update methods can be added later as needed for admin
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
