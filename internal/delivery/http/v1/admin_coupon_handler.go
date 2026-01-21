package v1

import (
	"encoding/json"
	"net/http"
	"rokomferi-backend/internal/usecase"
	"strconv"
)

// AdminCouponHandler handles admin coupon management endpoints.
// L9: Thin handler layer - delegates all logic to usecase.
type AdminCouponHandler struct {
	couponUC *usecase.CouponUsecase
}

// NewAdminCouponHandler creates a new AdminCouponHandler.
func NewAdminCouponHandler(uc *usecase.CouponUsecase) *AdminCouponHandler {
	return &AdminCouponHandler{couponUC: uc}
}

// ListCoupons returns paginated list of all coupons.
// GET /api/v1/admin/coupons?page=1&limit=20
func (h *AdminCouponHandler) ListCoupons(w http.ResponseWriter, r *http.Request) {
	limit := 20
	page := 1

	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 0 {
		page = p
	}
	offset := (page - 1) * limit

	coupons, total, err := h.couponUC.ListCoupons(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":  coupons,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// CreateCoupon creates a new coupon.
// POST /api/v1/admin/coupons
func (h *AdminCouponHandler) CreateCoupon(w http.ResponseWriter, r *http.Request) {
	var req usecase.CreateCouponRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	coupon, err := h.couponUC.CreateCoupon(r.Context(), req)
	if err != nil {
		// L9: Return 400 for validation errors, 500 for system errors
		if isValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(coupon)
}

// GetCoupon returns a single coupon by ID.
// GET /api/v1/admin/coupons/{id}
func (h *AdminCouponHandler) GetCoupon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Coupon ID required", http.StatusBadRequest)
		return
	}

	coupon, err := h.couponUC.GetCoupon(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(coupon)
}

// UpdateCoupon updates an existing coupon.
// PUT /api/v1/admin/coupons/{id}
func (h *AdminCouponHandler) UpdateCoupon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Coupon ID required", http.StatusBadRequest)
		return
	}

	var req usecase.UpdateCouponRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.couponUC.UpdateCoupon(r.Context(), id, req); err != nil {
		if isValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// DeleteCoupon deletes a coupon by ID.
// DELETE /api/v1/admin/coupons/{id}
func (h *AdminCouponHandler) DeleteCoupon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "Coupon ID required", http.StatusBadRequest)
		return
	}

	if err := h.couponUC.DeleteCoupon(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// isValidationError checks if an error is a validation error based on message.
// L9: Simple heuristic - production would use typed errors.
func isValidationError(err error) bool {
	msg := err.Error()
	validationPhrases := []string{
		"is required",
		"must be",
		"cannot exceed",
		"already exists",
		"not found",
		"invalid",
	}
	for _, phrase := range validationPhrases {
		if stringContains(msg, phrase) {
			return true
		}
	}
	return false
}

// stringContains is a simple substring check helper.
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
