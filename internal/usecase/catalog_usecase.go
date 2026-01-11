package usecase

import (
	"context"
	"rokomferi-backend/internal/domain"
)

type CatalogUsecase struct {
	productRepo domain.ProductRepository
}

func NewCatalogUsecase(repo domain.ProductRepository) *CatalogUsecase {
	return &CatalogUsecase{productRepo: repo}
}

func (u *CatalogUsecase) GetCategoryTree(ctx context.Context) ([]domain.Category, error) {
	return u.productRepo.GetCategoryTree(ctx)
}

func (u *CatalogUsecase) ListProducts(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	// Add business logic here if needed (e.g., validate filters)
	return u.productRepo.GetProducts(ctx, filter)
}

func (u *CatalogUsecase) GetProductDetails(ctx context.Context, slug string) (*domain.Product, error) {
	return u.productRepo.GetProductBySlug(ctx, slug)
}
