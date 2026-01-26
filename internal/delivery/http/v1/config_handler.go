package v1

import (
	"encoding/json"
	"net/http"
	"rokomferi-backend/internal/domain"
	"rokomferi-backend/pkg/cache"
	"time"
)

type ConfigHandler struct {
	cache cache.CacheService
}

func NewConfigHandler(cache cache.CacheService) *ConfigHandler {
	return &ConfigHandler{cache: cache}
}

// GET /api/v1/config/enums
func (h *ConfigHandler) GetEnums(w http.ResponseWriter, r *http.Request) {
	// Cache Key
	cacheKey := "system:config:enums"

	// Check Cache
	if val, found := h.cache.Get(cacheKey); found {
		w.Header().Set("Content-Type", "application/json")
		// Start Cache Headers
		w.Header().Set("Cache-Control", "public, max-age=3600")
		json.NewEncoder(w).Encode(val)
		return
	}

	response := map[string]interface{}{
		"orderStatuses":   domain.OrderStatuses,
		"paymentStatuses": domain.PaymentStatuses,
		"paymentMethods":  domain.PaymentMethods,
	}

	h.cache.Set(cacheKey, response, 1*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(response)
}
