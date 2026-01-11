package v1

import (
	"encoding/json"
	"net/http"
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
	userID := r.Context().Value("userID").(string)
	cart, err := h.orderUC.GetMyCart(r.Context(), userID)
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
	userID := r.Context().Value("userID").(string)
	var req addToCartReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	cart, err := h.orderUC.AddToCart(r.Context(), userID, req.ProductID, req.Quantity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cart)
}

// --- Order Handlers ---

func (h *OrderHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	var req usecase.CheckoutReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	order, err := h.orderUC.Checkout(r.Context(), userID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // 400 if cart empty, 500 otherwise. Simplified for now.
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) GetMyOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	orders, err := h.orderUC.GetMyOrders(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
