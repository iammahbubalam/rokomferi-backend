package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"rokomferi-backend/internal/domain"
	"rokomferi-backend/pkg/utils"
	"time"
)

type AuthUsecase struct {
	userRepo domain.UserRepository
	clientID string
}

func NewAuthUsecase(userRepo domain.UserRepository, clientID string) *AuthUsecase {
	return &AuthUsecase{
		userRepo: userRepo,
		clientID: clientID,
	}
}

type GoogleUser struct {
	ID            string      `json:"sub"`
	Email         string      `json:"email"`
	EmailVerified interface{} `json:"email_verified"` // Can be string ("true") or bool (true)
	Name          string      `json:"name"`
	GivenName     string      `json:"given_name"`
	FamilyName    string      `json:"family_name"`
	Picture       string      `json:"picture"`
	Aud           string      `json:"aud"`
}

func (u *AuthUsecase) AuthenticateGoogle(ctx context.Context, idToken string) (string, *domain.User, error) {
	slog.Info("Authenticating with Google", "token_length", len(idToken))
	// 1. Verify Google Token manually
	userInfo, err := verifyGoogleToken(idToken)
	if err != nil {
		slog.Error("Google token verification failed", "error", err)
		return "", nil, fmt.Errorf("invalid google token: %v", err)
	}

	// In production, verify Audience matches Client ID
	if userInfo.Aud != u.clientID {
		return "", nil, fmt.Errorf("token audience mismatch: expected %s, got %s", u.clientID, userInfo.Aud)
	}

	// 2. Find or Create User
	user, err := u.userRepo.GetByEmail(ctx, userInfo.Email)
	if err != nil {
		return "", nil, err
	}

	if user == nil {
		slog.Info("Creating new user", "email", userInfo.Email)
		// Create new user
		user = &domain.User{
			ID:        fmt.Sprintf("u_%d", time.Now().UnixNano()), // Simple ID generation
			Email:     userInfo.Email,
			FirstName: userInfo.GivenName,
			LastName:  userInfo.FamilyName,
			Avatar:    userInfo.Picture,
			Role:      "customer",
		}
		if err := u.userRepo.Create(ctx, user); err != nil {
			slog.Error("Failed to create user", "error", err)
			return "", nil, err
		}
	} else {
		slog.Info("Existing user found", "user_id", user.ID)
	}

	// 3. Generate JWT
	token, err := utils.GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

func verifyGoogleToken(token string) (*GoogleUser, error) {
	// We use access_token because frontend useGoogleLogin returns access_token by default for custom buttons.
	// https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=...
	resp, err := http.Get(fmt.Sprintf("https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=%s", token))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Error("Google API returned error", "status", resp.StatusCode)
		return nil, fmt.Errorf("google api returned status: %d", resp.StatusCode)
	}

	var gUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		slog.Error("Failed to decode Google user info", "error", err)
		return nil, err
	}
	slog.Info("Google token verified successfully", "email", gUser.Email)

	return &gUser, nil
}

func (u *AuthUsecase) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return u.userRepo.GetByID(ctx, id)
}
