package postgres

import (
	"context"
	"fmt"
	"rokomferi-backend/internal/domain"
	"time"

	"gorm.io/gorm"
)

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	db.AutoMigrate(&domain.Category{}, &domain.Product{}, &domain.Variant{}, &domain.InventoryLog{})
	return &productRepository{db: db}
}

func (r *productRepository) GetCategoryTree(ctx context.Context) ([]domain.Category, error) {
	var categories []domain.Category
	// Fetch only top-level categories, eagerly load children
	// Note: GORM preload only goes one level deep by default unless recursive query is used or simple Preload("Children").
	// For unlimited depth, we might need CTE, but for simple menu:
	if err := r.db.WithContext(ctx).Where("parent_id IS NULL").Preload("Children").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *productRepository) GetProducts(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	var products []domain.Product
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Product{})

	if filter.CategorySlug != "" {
		// Join categories to filter by slug
		query = query.Joins("JOIN categories ON categories.id = products.category_id").
			Where("categories.slug = ?", filter.CategorySlug)
	}

	if filter.Query != "" {
		query = query.Where("products.name ILIKE ?", "%"+filter.Query+"%")
	}

	if filter.MinPrice > 0 {
		query = query.Where("base_price >= ?", filter.MinPrice)
	}
	if filter.MaxPrice > 0 {
		query = query.Where("base_price <= ?", filter.MaxPrice)
	}

	// Count total before pagination
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Sorting
	switch filter.Sort {
	case "price_asc":
		query = query.Order("base_price ASC")
	case "price_desc":
		query = query.Order("base_price DESC")
	default:
		query = query.Order("created_at DESC")
	}

	// Pagination
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100 // Enforce hard limit
	}
	query = query.Limit(filter.Limit).Offset(filter.Offset)

	if err := query.Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// UpdateStock manages inventory with a strict Audit Log.
func (r *productRepository) UpdateStock(ctx context.Context, productID string, quantity int, reason, referenceID string) error {
	db := getDB(ctx, r.db)

	return db.Transaction(func(tx *gorm.DB) error {
		// 1. Update Product Stock
		// If deducting (negative quantity), ensure we don't go below 0
		var result *gorm.DB
		if quantity < 0 {
			result = tx.Model(&domain.Product{}).
				Where("id = ? AND stock >= ?", productID, -quantity).
				UpdateColumn("stock", gorm.Expr("stock + ?", quantity))
		} else {
			result = tx.Model(&domain.Product{}).
				Where("id = ?", productID).
				UpdateColumn("stock", gorm.Expr("stock + ?", quantity))
		}

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("insufficient stock or product not found: %s", productID)
		}

		// 2. Create Audit Log
		logEntry := domain.InventoryLog{
			ProductID:    productID,
			ChangeAmount: quantity,
			Reason:       reason,
			ReferenceID:  referenceID,
			CreatedAt:    time.Now(),
		}
		if err := tx.Create(&logEntry).Error; err != nil {
			return fmt.Errorf("failed to create inventory log: %v", err)
		}

		return nil
	})
}

func (r *productRepository) GetProductBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	var product domain.Product
	if err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		Preload("Variants").
		Preload("Category").
		First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *productRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	var product domain.Product
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Preload("Variants").
		Preload("Category").
		First(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

// --- Admin Implementation ---

func (r *productRepository) CreateProduct(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *productRepository) UpdateProduct(ctx context.Context, product *domain.Product) error {
	// Fix: Use Select to ensure zero values (like IsActive=false) are updated.
	// We exclude ID, CSV, SKU, Slug, CreatedAt from arbitrary updates here (safety).
	return r.db.WithContext(ctx).Model(product).
		Select("Name", "Description", "BasePrice", "SalePrice", "Stock", "StockStatus", "LowStockThreshold", "IsFeatured", "IsActive", "CategoryID", "Media", "Attributes", "Specs", "UpdatedAt").
		Updates(product).Error
}

func (r *productRepository) DeleteProduct(ctx context.Context, id string) error {
	// Soft delete if domain.Product has DeletedAt, otherwise Hard Delete.
	// Since we defined simple struct, this is Hard Delete unless we add gorm.Model or DeletedAt.
	// For "Robustness", checking constraints (orders, stock) is Wise.
	// But standard DELETE for now.
	return r.db.WithContext(ctx).Delete(&domain.Product{}, "id = ?", id).Error
}
