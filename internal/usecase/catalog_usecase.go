package usecase

import (
	"context"
	"fmt"
	"rokomferi-backend/internal/domain"
	"rokomferi-backend/pkg/utils"
	"time"
)

type CatalogUsecase struct {
	repo domain.ProductRepository
}

func NewCatalogUsecase(repo domain.ProductRepository) *CatalogUsecase {
	return &CatalogUsecase{repo: repo}
}

func (uc *CatalogUsecase) CreateProduct(ctx context.Context, product *domain.Product) error {
	// 1. Generate Slug if missing
	if product.Slug == "" {
		product.Slug = utils.GenerateSlug(product.Name)
	}
	// 2. Set Defaults
	if product.SKU == "" {
		return fmt.Errorf("SKU is required")
	}
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()
	product.IsActive = true

	return uc.repo.CreateProduct(ctx, product)
}

func (uc *CatalogUsecase) UpdateProduct(ctx context.Context, product *domain.Product) error {
	product.UpdatedAt = time.Now()
	// Prevent slug update? Or allow re-generation? Let's allow simple update for now.
	return uc.repo.UpdateProduct(ctx, product)
}

func (uc *CatalogUsecase) DeleteProduct(ctx context.Context, id string) error {
	return uc.repo.DeleteProduct(ctx, id)
}

func (uc *CatalogUsecase) AdjustStock(ctx context.Context, productID string, changeAmount int, reason, referenceID string) error {
	return uc.repo.UpdateStock(ctx, productID, changeAmount, reason, referenceID)
}

func (u *CatalogUsecase) GetCategoryTree(ctx context.Context) ([]domain.Category, error) {
	return u.repo.GetCategoryTree(ctx)
}

func (u *CatalogUsecase) ListProducts(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	// Add business logic here if needed (e.g., validate filters)
	return u.repo.GetProducts(ctx, filter)
}

func (u *CatalogUsecase) GetProductDetails(ctx context.Context, slug string) (*domain.Product, error) {
	return u.repo.GetProductBySlug(ctx, slug)
}
