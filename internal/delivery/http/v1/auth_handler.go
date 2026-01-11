package v1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"rokomferi-backend/internal/domain"
	"rokomferi-backend/internal/usecase"
)

type AuthHandler struct {
	authUC *usecase.AuthUsecase
}

func NewAuthHandler(authUC *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUC: authUC}
}

type googleLoginReq struct {
	IDToken string `json:"idToken"`
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	slog.Info("GoogleLogin request received")
	var req struct {
		IDToken string `json:"idToken"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Updated call to AuthenticateGoogle
	accessToken, refreshToken, user, err := h.authUC.AuthenticateGoogle(r.Context(), req.IDToken)
	if err != nil {
		slog.Error("Authentication failed", "error", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Set Refresh Token as HttpOnly Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/", // Allows access for refresh endpoint
		HttpOnly: true,
		Secure:   true, // Should be true in production/https
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	slog.Info("User authenticated successfully", "user_id", user.ID, "email", user.Email)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accessToken": accessToken,
		"user":        user,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "Refresh token missing", http.StatusUnauthorized)
		return
	}

	newAccessToken, err := h.authUC.RefreshAccessToken(r.Context(), cookie.Value)
	if err != nil {
		slog.Error("Token refresh failed", "error", err)
		// Clear cookie if invalid
		http.SetCookie(w, &http.Cookie{Name: "refresh_token", MaxAge: -1, Path: "/"})
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accessToken": newAccessToken,
	})
}

// --- Address Handlers ---

func (h *AuthHandler) AddAddress(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	var req domain.Address
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	addr, err := h.authUC.AddAddress(r.Context(), userID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(addr)
}

func (h *AuthHandler) GetAddresses(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(string)
	addrs, err := h.authUC.GetAddresses(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(addrs)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	// Assumes AuthMiddleware has run and set UserID in context
	// We might need to fetch fresh user data from DB or just return claims
	// For API needs 2.2, we return ID, Email, Role, Preferences

	userID, ok := r.Context().Value("userID").(string) // Needs to match middleware key type, better to use exported key
	if !ok {
		// Ideally middleware handles this, but safe check
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Delegate to Usecase to get full user profile if needed
	user, err := h.authUC.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:   "refresh_token",
		MaxAge: -1,
		Path:   "/",
	})
	w.WriteHeader(http.StatusOK)
}
