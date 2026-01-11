package v1

import (
	"encoding/json"
	"net/http"
	"rokomferi-backend/internal/usecase"
	"strconv"
)

type AdminOrderHandler struct {
	orderUC *usecase.OrderUsecase
}

func NewAdminOrderHandler(uc *usecase.OrderUsecase) *AdminOrderHandler {
	return &AdminOrderHandler{orderUC: uc}
}

func (h *AdminOrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 20
	}
	status := r.URL.Query().Get("status")

	orders, total, err := h.orderUC.GetAllOrders(r.Context(), page, limit, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"orders": orders,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

func (h *AdminOrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Order ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	if err := h.orderUC.UpdateOrderStatus(r.Context(), id, req.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
