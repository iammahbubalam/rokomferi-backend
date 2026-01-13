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

func (uc *CatalogUsecase) CreateCategory(ctx context.Context, category *domain.Category) error {
	if category.Name == "" {
		return fmt.Errorf("category name is required")
	}
	if category.Slug == "" {
		category.Slug = utils.GenerateSlug(category.Name)
	}
	if category.ID == "" {
		category.ID = utils.GenerateUUID()
	}
	// Ensure defaults
	if !category.IsActive {
		category.IsActive = true
	}
	return uc.repo.CreateCategory(ctx, category)
}

func (uc *CatalogUsecase) UpdateCategory(ctx context.Context, category *domain.Category) error {
	if category.ID == "" {
		return fmt.Errorf("category ID required")
	}
	return uc.repo.UpdateCategory(ctx, category)
}

func (uc *CatalogUsecase) DeleteCategory(ctx context.Context, id string) error {
	return uc.repo.DeleteCategory(ctx, id)
}

func (uc *CatalogUsecase) ReorderCategories(ctx context.Context, updates []domain.CategoryReorderItem) error {
	return uc.repo.ReorderCategories(ctx, updates)
}

func (u *CatalogUsecase) ListProducts(ctx context.Context, filter domain.ProductFilter) ([]domain.Product, int64, error) {
	// Add business logic here if needed (e.g., validate filters)
	return u.repo.GetProducts(ctx, filter)
}

func (u *CatalogUsecase) GetProductDetails(ctx context.Context, slug string) (*domain.Product, error) {
	return u.repo.GetProductBySlug(ctx, slug)
}

func (u *CatalogUsecase) AddReview(ctx context.Context, userID, productID string, rating int, comment string) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	review := &domain.Review{
		ID:        utils.GenerateUUID(),
		UserID:    userID,
		ProductID: productID,
		Rating:    rating,
		Comment:   comment,
		CreatedAt: time.Now(),
	}

	return u.repo.CreateReview(ctx, review)
}

func (u *CatalogUsecase) GetProductReviews(ctx context.Context, productID string) ([]domain.Review, error) {
	return u.repo.GetReviews(ctx, productID)
}

// --- Collections ---

func (uc *CatalogUsecase) GetCollections(ctx context.Context) ([]domain.Collection, error) {
	return uc.repo.GetCollections(ctx)
}

func (uc *CatalogUsecase) GetCollectionBySlug(ctx context.Context, slug string) (*domain.Collection, error) {
	return uc.repo.GetCollectionBySlug(ctx, slug)
}

func (uc *CatalogUsecase) CreateCollection(ctx context.Context, collection *domain.Collection) error {
	if collection.Title == "" {
		return fmt.Errorf("collection title is required")
	}
	if collection.Slug == "" {
		collection.Slug = utils.GenerateSlug(collection.Title)
	}
	if collection.ID == "" {
		collection.ID = utils.GenerateUUID()
	}
	if !collection.IsActive {
		collection.IsActive = true
	}
	collection.CreatedAt = time.Now()
	collection.UpdatedAt = time.Now()
	return uc.repo.CreateCollection(ctx, collection)
}

func (uc *CatalogUsecase) UpdateCollection(ctx context.Context, collection *domain.Collection) error {
	collection.UpdatedAt = time.Now()
	return uc.repo.UpdateCollection(ctx, collection)
}

func (uc *CatalogUsecase) DeleteCollection(ctx context.Context, id string) error {
	return uc.repo.DeleteCollection(ctx, id)
}

func (uc *CatalogUsecase) AddProductToCollection(ctx context.Context, collectionID, productID string) error {
	return uc.repo.AddProductToCollection(ctx, collectionID, productID)
}

func (uc *CatalogUsecase) RemoveProductFromCollection(ctx context.Context, collectionID, productID string) error {
	return uc.repo.RemoveProductFromCollection(ctx, collectionID, productID)
}
