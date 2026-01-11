package v1

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"rokomferi-backend/internal/usecase"
	"time"
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
	var req googleLoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode GoogleLogin request", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, user, err := h.authUC.AuthenticateGoogle(r.Context(), req.IDToken)
	if err != nil {
		slog.Error("Authentication failed", "error", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	slog.Info("User authenticated successfully", "user_id", user.ID, "email", user.Email)

	// Set Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    token,
		HttpOnly: true,
		Secure:   true, // Should be true in prod
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accessToken": token,
		"user":        user,
	})
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
	// Clear Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
	})
	w.WriteHeader(http.StatusNoContent)
}
