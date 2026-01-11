package postgres

import (
	"context"
	"rokomferi-backend/internal/domain"

	"gorm.io/gorm"
)

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	db.AutoMigrate(&domain.Category{}, &domain.Product{}, &domain.Variant{})
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
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	if err := query.Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
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
