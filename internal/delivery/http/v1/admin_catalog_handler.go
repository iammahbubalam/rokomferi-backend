package v1

import (
	"encoding/json"
	"net/http"
	"rokomferi-backend/internal/domain"
	"rokomferi-backend/internal/usecase"
)

type AdminCatalogHandler struct {
	catalogUC *usecase.CatalogUsecase
}

func NewAdminCatalogHandler(uc *usecase.CatalogUsecase) *AdminCatalogHandler {
	return &AdminCatalogHandler{catalogUC: uc}
}

type productReq struct {
	domain.Product
	CategoryIDs []string `json:"categoryIds"`
}

func (h *AdminCatalogHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req productReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	product := req.Product
	// Map CategoryIDs to Categories
	if len(req.CategoryIDs) > 0 {
		product.Categories = make([]domain.Category, len(req.CategoryIDs))
		for i, id := range req.CategoryIDs {
			product.Categories[i] = domain.Category{ID: id}
		}
	}

	if err := h.catalogUC.CreateProduct(r.Context(), &product); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (h *AdminCatalogHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	var req productReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	product := req.Product
	product.ID = id

	// Map CategoryIDs to Categories
	if len(req.CategoryIDs) > 0 {
		product.Categories = make([]domain.Category, len(req.CategoryIDs))
		for i, id := range req.CategoryIDs {
			product.Categories[i] = domain.Category{ID: id}
		}
	} else if req.CategoryIDs != nil {
		// Explicit empty array clears categories
		product.Categories = []domain.Category{}
	}

	if err := h.catalogUC.UpdateProduct(r.Context(), &product); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *AdminCatalogHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.DeleteProduct(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

type adjustStockReq struct {
	ProductID    string `json:"productId"`
	ChangeAmount int    `json:"changeAmount"` // negative to deduct
	Reason       string `json:"reason"`
}

func (h *AdminCatalogHandler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	// Require Admin (Handled by middleware)
	// Get ID from authenticated user for reference (optional)
	// For now, we just take the body.

	var req adjustStockReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Get Admin ID from context
	adminUser, _ := r.Context().Value(domain.UserContextKey).(*domain.User)
	referenceID := "admin"
	if adminUser != nil {
		referenceID = adminUser.ID
	}

	if err := h.catalogUC.AdjustStock(r.Context(), req.ProductID, req.ChangeAmount, req.Reason, referenceID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "stock updated"})
}

func (h *AdminCatalogHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var category domain.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.CreateCategory(r.Context(), &category); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(category)
}

func (h *AdminCatalogHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Category ID required", http.StatusBadRequest)
		return
	}

	var category domain.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	category.ID = id

	if err := h.catalogUC.UpdateCategory(r.Context(), &category); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *AdminCatalogHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Category ID required", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.DeleteCategory(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

type reorderReq struct {
	Updates []domain.CategoryReorderItem `json:"updates"`
}

func (h *AdminCatalogHandler) ReorderCategories(w http.ResponseWriter, r *http.Request) {
	var req reorderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.ReorderCategories(r.Context(), req.Updates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reordered"})
}

// --- Collections ---

func (h *AdminCatalogHandler) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var collection domain.Collection
	if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.CreateCollection(r.Context(), &collection); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(collection)
}

func (h *AdminCatalogHandler) UpdateCollection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	var collection domain.Collection
	if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	collection.ID = id

	if err := h.catalogUC.UpdateCollection(r.Context(), &collection); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *AdminCatalogHandler) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Collection ID required", http.StatusBadRequest)
		return
	}

	if err := h.catalogUC.DeleteCollection(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (h *AdminCatalogHandler) ManageCollectionProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Collection ID
	var req struct {
		ProductID string `json:"productId"`
		Action    string `json:"action"` // "add" or "remove"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var err error
	if req.Action == "add" {
		err = h.catalogUC.AddProductToCollection(r.Context(), id, req.ProductID)
	} else {
		err = h.catalogUC.RemoveProductFromCollection(r.Context(), id, req.ProductID)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
