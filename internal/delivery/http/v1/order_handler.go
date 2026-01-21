package v1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"rokomferi-backend/internal/domain"
	"rokomferi-backend/internal/usecase"
)

type OrderHandler struct {
	orderUC *usecase.OrderUsecase
}

func NewOrderHandler(uc *usecase.OrderUsecase) *OrderHandler {
	return &OrderHandler{orderUC: uc}
}

// --- Cart Handlers ---

func (h *OrderHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	cart, err := h.orderUC.GetMyCart(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

type addToCartReq struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

func (h *OrderHandler) AddToCart(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req addToCartReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	cart, err := h.orderUC.AddToCart(r.Context(), user.ID, req.ProductID, req.Quantity)
	if err != nil {
		slog.Error("AddToCart failed", "user_id", user.ID, "product_id", req.ProductID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

func (h *OrderHandler) RemoveFromCart(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	productID := r.PathValue("productId")
	if productID == "" {
		http.Error(w, "Product ID required", http.StatusBadRequest)
		return
	}

	cart, err := h.orderUC.RemoveFromCart(r.Context(), user.ID, productID)
	if err != nil {
		slog.Error("RemoveFromCart failed", "user_id", user.ID, "product_id", productID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

func (h *OrderHandler) UpdateCart(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ProductID string `json:"productId"`
		Quantity  int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	cart, err := h.orderUC.UpdateCartItemQuantity(r.Context(), user.ID, req.ProductID, req.Quantity)
	if err != nil {
		slog.Error("UpdateCart failed", "user_id", user.ID, "product_id", req.ProductID, "error", err)

		statusCode := http.StatusInternalServerError
		if contains(err.Error(), "insufficient stock") || contains(err.Error(), "out of stock") || contains(err.Error(), "not found") {
			statusCode = http.StatusBadRequest
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"message": err.Error(),
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

// --- Order Handlers ---

func (h *OrderHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req usecase.CheckoutReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	order, err := h.orderUC.Checkout(r.Context(), user.ID, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		errMsg := err.Error()
		statusCode := http.StatusInternalServerError

		if contains(errMsg, "insufficient stock") || contains(errMsg, "out of stock") || contains(errMsg, "cart is empty") || contains(errMsg, "not found") {
			statusCode = http.StatusBadRequest
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"message": errMsg,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) GetMyOrders(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(domain.UserContextKey).(*domain.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	orders, err := h.orderUC.GetMyOrders(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[0:len(substr)] == substr || contains(s[1:], substr)))
}
