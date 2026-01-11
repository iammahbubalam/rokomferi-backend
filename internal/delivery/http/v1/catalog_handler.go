package v1

import (
	"encoding/json"
	"net/http"
	"rokomferi-backend/internal/domain"
	"rokomferi-backend/internal/usecase"
	"strconv"
)

type CatalogHandler struct {
	catalogUC *usecase.CatalogUsecase
}

func NewCatalogHandler(uc *usecase.CatalogUsecase) *CatalogHandler {
	return &CatalogHandler{catalogUC: uc}
}

func (h *CatalogHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.catalogUC.GetCategoryTree(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cats)
}

func (h *CatalogHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit == 0 {
		limit = 20
	}
	page, _ := strconv.Atoi(query.Get("page"))
	if page == 0 {
		page = 1
	}

	minPrice, _ := strconv.ParseFloat(query.Get("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(query.Get("max_price"), 64)

	filter := domain.ProductFilter{
		CategorySlug: query.Get("category_slug"),
		Query:        query.Get("q"),
		Sort:         query.Get("sort"),
		MinPrice:     minPrice,
		MaxPrice:     maxPrice,
		Limit:        limit,
		Offset:       (page - 1) * limit,
	}

	products, total, err := h.catalogUC.ListProducts(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": products,
		"pagination": map[string]interface{}{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func (h *CatalogHandler) GetProductDetails(w http.ResponseWriter, r *http.Request) {
	// Simple Slug extraction - in standard mux with Go 1.22 we can use PathValue
	// But let's assume standard behavior: /products/{slug}
	// Note: We need to register this correctly in mux

	slug := r.PathValue("slug")
	if slug == "" {
		http.Error(w, "Slug required", http.StatusBadRequest)
		return
	}

	product, err := h.catalogUC.GetProductDetails(r.Context(), slug)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}
