package postgres

import (
	"context"
	"fmt"
	"rokomferi-backend/internal/domain"
	"strings"
	"time"

	"gorm.io/gorm"
)

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	// WARNING: Disabled for performance.
	// db.AutoMigrate(&domain.Category{}, &domain.Product{}, &domain.Variant{}, &domain.InventoryLog{}, &domain.Review{}, &domain.Collection{})
	return &productRepository{db: db}
}

func (r *productRepository) GetCategoryTree(ctx context.Context) ([]domain.Category, error) {
	var categories []domain.Category
	// Return ALL root categories with nested children (recursive preload)
	if err := r.db.WithContext(ctx).
		Where("parent_id IS NULL").
		Order("order_index asc").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index asc")
		}).
		Preload("Children.Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index asc")
		}).
		Preload("Children.Children.Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_index asc")
		}).
		Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// GetNavCategoryTree returns only active+nav categories for public navbar
func (r *productRepository) GetNavCategoryTree(ctx context.Context) ([]domain.Category, error) {
	var categories []domain.Category
	if err := r.db.WithContext(ctx).
		Where("parent_id IS NULL AND is_active = ? AND show_in_nav = ?", true, true).
		Order("order_index asc").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ? AND show_in_nav = ?", true, true).Order("order_index asc")
		}).
		Preload("Children.Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ? AND show_in_nav = ?", true, true).Order("order_index asc")
		}).
		Preload("Children.Children.Children", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ? AND show_in_nav = ?", true, true).Order("order_index asc")
		}).
		Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *productRepository) GetCategoryBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	var category domain.Category
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&category).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

func (r *productRepository) CreateCategory(ctx context.Context, category *domain.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *productRepository) UpdateCategory(ctx context.Context, category *domain.Category) error {
	// Save will insert if no ID or update all fields if ID exists
	return r.db.WithContext(ctx).Save(category).Error
}

func (r *productRepository) DeleteCategory(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Category{}, "id = ?", id).Error
}

func (r *productRepository) ReorderCategories(ctx context.Context, updates []domain.CategoryReorderItem) error {
	if len(updates) == 0 {
		return nil
	}

	// Batch Update using CASE statements to avoid N+1 DB calls
	var orderArgs []interface{}
	var parentArgs []interface{}
	var whereArgs []interface{}
	var ids []string

	orderCase := strings.Builder{}
	orderCase.WriteString("CASE ")

	parentCase := strings.Builder{}
	parentCase.WriteString("CASE ")

	for _, item := range updates {
		orderCase.WriteString("WHEN id = ? THEN ? ")
		orderArgs = append(orderArgs, item.ID, item.OrderIndex)

		parentCase.WriteString("WHEN id = ? THEN ? ")
		parentArgs = append(parentArgs, item.ID, item.ParentID)

		ids = append(ids, "?")
		whereArgs = append(whereArgs, item.ID)
	}

	orderCase.WriteString("ELSE order_index END")
	parentCase.WriteString("ELSE parent_id END")

	query := fmt.Sprintf("UPDATE categories SET order_index = %s, parent_id = %s WHERE id IN (%s)",
		orderCase.String(), parentCase.String(), strings.Join(ids, ","))

	var allArgs []interface{}
	allArgs = append(allArgs, orderArgs...)
	allArgs = append(allArgs, parentArgs...)
	allArgs = append(allArgs, whereArgs...)

	return r.db.WithContext(ctx).Exec(query, allArgs...).Error
}

func (r *productRepository) GetProducts(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	var products []domain.Product
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Product{}).Preload("Categories")

	if filter.CategorySlug != "" {
		// Join categories to filter by slug
		query = query.Joins("JOIN product_categories ON product_categories.product_id = products.id").
			Joins("JOIN categories ON categories.id = product_categories.category_id").
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

	// Count total before pagination (distinct needed because of joins?)
	// GORM Count with Joins usually counts rows. With Many-to-Many + Join, we might get duplicates if a product is in multiple categories matching criteria?
	// But we filter by specific category slug, so 1 row per product for THAT category.
	if err := query.Group("products.id").Count(&total).Error; err != nil {
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
		Preload("Categories").
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
		Preload("Categories").
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
	err := r.db.WithContext(ctx).Model(product).
		Select("Name", "Description", "BasePrice", "SalePrice", "Stock", "StockStatus", "LowStockThreshold", "IsFeatured", "IsActive", "Media", "Attributes", "Specs", "UpdatedAt").
		Updates(product).Error

	if err != nil {
		return err
	}

	// Update Categories Association
	return r.db.Model(product).Association("Categories").Replace(product.Categories)
}

func (r *productRepository) DeleteProduct(ctx context.Context, id string) error {
	// Soft delete if domain.Product has DeletedAt, otherwise Hard Delete.
	// Since we defined simple struct, this is Hard Delete unless we add gorm.Model or DeletedAt.
	// For "Robustness", checking constraints (orders, stock) is Wise.
	// But standard DELETE for now.
	return r.db.WithContext(ctx).Delete(&domain.Product{}, "id = ?", id).Error
}

// --- Reviews ---

func (r *productRepository) CreateReview(ctx context.Context, review *domain.Review) error {
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *productRepository) GetReviews(ctx context.Context, productID string) ([]domain.Review, error) {
	var reviews []domain.Review
	err := r.db.WithContext(ctx).
		Where("product_id = ?", productID).
		Preload("User"). // Eager load reviewer info
		Order("created_at desc").
		Find(&reviews).Error
	return reviews, err
}

// --- Collections ---

func (r *productRepository) GetCollections(ctx context.Context) ([]domain.Collection, error) {
	var collections []domain.Collection
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("created_at desc").
		Find(&collections).Error
	return collections, err
}

// Fix Collection Preload
func (r *productRepository) GetCollectionBySlug(ctx context.Context, slug string) (*domain.Collection, error) {
	var collection domain.Collection
	// Preload Products and their Media to display images on frontend
	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		Preload("Products", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", true).Order("created_at desc")
		}).
		Preload("Products.Categories"). // Updated from Products.Category
		First(&collection).Error
	return &collection, err
}

func (r *productRepository) CreateCollection(ctx context.Context, collection *domain.Collection) error {
	return r.db.WithContext(ctx).Create(collection).Error
}

func (r *productRepository) UpdateCollection(ctx context.Context, collection *domain.Collection) error {
	return r.db.WithContext(ctx).Model(collection).
		Select("Title", "Slug", "Description", "Image", "Story", "IsActive", "UpdatedAt").
		Updates(collection).Error
}

func (r *productRepository) DeleteCollection(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Collection{}, "id = ?", id).Error
}

func (r *productRepository) AddProductToCollection(ctx context.Context, collectionID, productID string) error {
	collection := domain.Collection{ID: collectionID}
	product := domain.Product{ID: productID}
	return r.db.WithContext(ctx).Model(&collection).Association("Products").Append(&product)
}

func (r *productRepository) RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error {
	collection := domain.Collection{ID: collectionID}
	product := domain.Product{ID: productID}
	return r.db.WithContext(ctx).Model(&collection).Association("Products").Delete(&product)
}
