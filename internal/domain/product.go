package domain

import (
	"context"
	"time"
)

// --- Interfaces ---

type TransactionManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Category struct {
	ID              string     `json:"id" gorm:"primaryKey"`
	Name            string     `json:"name"`
	Slug            string     `json:"slug" gorm:"uniqueIndex"`
	ParentID        *string    `json:"parentId"`
	Parent          *Category  `json:"-" gorm:"foreignKey:ParentID"`
	Children        []Category `json:"children" gorm:"foreignKey:ParentID"`
	OrderIndex      int        `json:"orderIndex" gorm:"default:0"`
	Icon            string     `json:"icon"`
	Image           string     `json:"image"`
	IsActive        bool       `json:"isActive" gorm:"default:true"`
	ShowInNav       bool       `json:"showInNav" gorm:"default:true"`
	MetaTitle       string     `json:"metaTitle"`
	MetaDescription string     `json:"metaDescription"`
	Keywords        string     `json:"keywords"`
	Products        []Product  `json:"products" gorm:"many2many:product_categories;"`
}

type CategoryReorderItem struct {
	ID         string  `json:"id"`
	ParentID   *string `json:"parentId"`
	OrderIndex int     `json:"orderIndex"`
}

type Product struct {
	ID                string       `json:"id" gorm:"primaryKey"`
	Name              string       `json:"name"`
	Slug              string       `json:"slug" gorm:"uniqueIndex"`
	SKU               string       `json:"sku" gorm:"uniqueIndex"` // Robust Inventory
	Description       string       `json:"description"`
	BasePrice         float64      `json:"basePrice"`
	SalePrice         *float64     `json:"salePrice"`
	Stock             int          `json:"stock" gorm:"default:0"` // Main inventory
	StockStatus       string       `json:"stockStatus"`
	LowStockThreshold int          `json:"lowStockThreshold" gorm:"default:5"`
	IsFeatured        bool         `json:"isFeatured"`
	IsActive          bool         `json:"isActive" gorm:"default:true"`
	Media             RawJSON      `json:"media" gorm:"type:jsonb"`
	Images            []string     `json:"images" gorm:"-"` // Mapped from Media
	Attributes        JSONB        `json:"attributes" gorm:"type:jsonb"`
	Specs             JSONB        `json:"specifications" gorm:"type:jsonb"`
	CreatedAt         time.Time    `json:"createdAt"`
	UpdatedAt         time.Time    `json:"updatedAt"`
	Variants          []Variant    `json:"variants" gorm:"foreignKey:ProductID"`
	Categories        []Category   `json:"categories" gorm:"many2many:product_categories;"`
	Collections       []Collection `json:"collections" gorm:"many2many:product_collections;"`
}

type Collection struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug" gorm:"uniqueIndex"`
	Description string    `json:"description"`
	Image       string    `json:"image"`
	Story       string    `json:"story"` // The rich text narrative
	IsActive    bool      `json:"isActive"`
	Products    []Product `json:"products" gorm:"many2many:product_collections;"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
	// Category Management
	GetCategoryTree(ctx context.Context) ([]Category, error)
	GetNavCategoryTree(ctx context.Context) ([]Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*Category, error)
	CreateCategory(ctx context.Context, category *Category) error
	UpdateCategory(ctx context.Context, category *Category) error
	DeleteCategory(ctx context.Context, id string) error
	ReorderCategories(ctx context.Context, updates []CategoryReorderItem) error

	// Collection Management
	GetCollections(ctx context.Context) ([]Collection, error)
	GetAllCollections(ctx context.Context) ([]Collection, error)
	GetCollectionBySlug(ctx context.Context, slug string) (*Collection, error)
	CreateCollection(ctx context.Context, collection *Collection) error
	UpdateCollection(ctx context.Context, collection *Collection) error
	DeleteCollection(ctx context.Context, id string) error
	AddProductToCollection(ctx context.Context, collectionID, productID string) error
	RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error

	GetProducts(ctx context.Context, filter ProductFilter) ([]Product, int64, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	GetProductByID(ctx context.Context, id string) (*Product, error)
	UpdateStock(ctx context.Context, productID string, quantity int, reason, referenceID string) error
	GetInventoryLogs(ctx context.Context, productID string, limit, offset int) ([]InventoryLog, int64, error)

	// Admin Management
	CreateProduct(ctx context.Context, product *Product) error
	UpdateProduct(ctx context.Context, product *Product) error
	UpdateProductStatus(ctx context.Context, id string, isActive bool) error
	DeleteProduct(ctx context.Context, id string) error

	// Reviews
	CreateReview(ctx context.Context, review *Review) error
	GetReviews(ctx context.Context, productID string) ([]Review, error)
}

type Review struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	ProductID string    `json:"productId"`
	UserID    string    `json:"userId"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	Rating    int       `json:"rating"` // 1-5
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
}

type ProductFilter struct {
	CategorySlug string
	Query        string
	MinPrice     float64
	MaxPrice     float64
	Sort         string // newest, price_asc, price_desc
	Limit        int
	Offset       int
	IsActive     *bool // nil = all, true = active, false = inactive
}

// --- Custom Types moved to types.go ---
