package usecase

import (
	"context"
	"fmt"
	"rokomferi-backend/internal/domain"
	"time"
)

type SitemapItem struct {
	Loc        string
	LastMod    string
	ChangeFreq string
	Priority   float32
}

type SitemapUsecase struct {
	productRepo domain.ProductRepository
	baseURL     string
}

func NewSitemapUsecase(repo domain.ProductRepository, baseURL string) *SitemapUsecase {
	if baseURL == "" {
		// Should be handled by config, but strictly no hardcoded production domain here.
		// Fallback to empty string or a placeholder if really needed, but better to trust injection.
	}
	return &SitemapUsecase{
		productRepo: repo,
		baseURL:     baseURL,
	}
}

func (u *SitemapUsecase) GenerateSitemap(ctx context.Context) ([]SitemapItem, error) {
	var items []SitemapItem
	now := time.Now().Format("2006-01-02")

	// 1. Static Pages
	statics := []string{"", "/shop", "/collections", "/about", "/contact", "/login"} // Empty string for root
	for _, s := range statics {
		items = append(items, SitemapItem{
			Loc:        u.baseURL + s,
			LastMod:    now,
			ChangeFreq: "daily",
			Priority:   0.8,
		})
	}
	// Root has higher priority
	items[0].Priority = 1.0

	// 2. Products (Active only)
	isActive := true
	filter := domain.ProductFilter{
		Limit:    2000, // Reasonable limit for now
		Offset:   0,
		IsActive: &isActive,
	}
	products, _, err := u.productRepo.GetProducts(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	for _, p := range products {
		items = append(items, SitemapItem{
			Loc:        fmt.Sprintf("%s/product/%s", u.baseURL, p.Slug),
			LastMod:    p.UpdatedAt.Format("2006-01-02"),
			ChangeFreq: "weekly",
			Priority:   0.9,
		})
	}

	// 3. Categories
	categories, err := u.productRepo.GetCategoryTree(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}
	var flattenCats func([]domain.Category)
	flattenCats = func(cats []domain.Category) {
		for _, c := range cats {
			if c.IsActive {
				items = append(items, SitemapItem{
					Loc:        fmt.Sprintf("%s/category/%s", u.baseURL, c.Slug),
					LastMod:    now, // Categories don't track update time well in domain, use now
					ChangeFreq: "daily",
					Priority:   0.8,
				})
				if len(c.Children) > 0 {
					flattenCats(c.Children)
				}
			}
		}
	}
	flattenCats(categories)

	// 4. Collections
	collections, err := u.productRepo.GetCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch collections: %w", err)
	}
	for _, c := range collections {
		if c.IsActive {
			items = append(items, SitemapItem{
				Loc:        fmt.Sprintf("%s/collection/%s", u.baseURL, c.Slug),
				LastMod:    c.UpdatedAt.Format("2006-01-02"),
				ChangeFreq: "weekly",
				Priority:   0.8,
			})
		}
	}

	return items, nil
}
