package main

import (
	"log"
	"rokomferi-backend/config"
	"rokomferi-backend/internal/domain"
	postgresPkg "rokomferi-backend/pkg/postgres"

	"gorm.io/gorm"
)

func main() {
	cfg := config.LoadConfig()
	db, err := postgresPkg.NewClient(cfg.DBUrl)
	if err != nil {
		log.Fatal(err)
	}

	seedCategories(db)
	seedProducts(db)
}

func seedCategories(db *gorm.DB) {
	cats := []domain.Category{
		{ID: "cat_women", Name: "Women", Slug: "women"},
		{ID: "cat_men", Name: "Men", Slug: "men"},
	}
	for _, c := range cats {
		db.FirstOrCreate(&c, domain.Category{ID: c.ID})
	}

	subCats := []domain.Category{
		{ID: "cat_sarees", Name: "Sarees", Slug: "sarees", ParentID: stringPtr("cat_women")},
	}
	for _, c := range subCats {
		db.FirstOrCreate(&c, domain.Category{ID: c.ID})
	}
}

func seedProducts(db *gorm.DB) {
	p := domain.Product{
		ID:          "p_001",
		Name:        "Midnight Katan",
		Slug:        "midnight-katan",
		Description: "Elegant silk saree.",
		BasePrice:   12000,
		CategoryID:  "cat_sarees",
		StockStatus: "in_stock",
		IsFeatured:  true,
	}
	db.FirstOrCreate(&p, domain.Product{ID: p.ID})
}

func stringPtr(s string) *string {
	return &s
}
