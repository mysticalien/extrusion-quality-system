package http

import (
	"encoding/json"
	"log/slog"
	nethttp "net/http"

	authservice "extrusion-quality-system/internal/auth"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
)

type AuthHandler struct {
	logger         *slog.Logger
	userRepository storage.UserRepository
	tokenManager   *authservice.TokenManager
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type userResponse struct {
	ID       domain.UserID `json:"id"`
	Username string        `json:"username"`
	Role     domain.Role   `json:"role"`
}

func NewAuthHandler(
	logger *slog.Logger,
	userRepository storage.UserRepository,
	tokenManager *authservice.TokenManager,
) *AuthHandler {
	return &AuthHandler{
		logger:         logger,
		userRepository: userRepository,
		tokenManager:   tokenManager,
	}
}

func (h *AuthHandler) Login(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != nethttp.MethodPost {
		w.Header().Set("Allow", nethttp.MethodPost)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	defer r.Body.Close()

	var request loginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		w.WriteHeader(nethttp.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body"})
		return
	}

	if request.Username == "" || request.Password == "" {
		w.WriteHeader(nethttp.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "username and password are required"})
		return
	}

	user, found, err := h.userRepository.FindByUsername(r.Context(), request.Username)
	if err != nil {
		h.logger.Error("find user by username failed", "error", err, "username", request.Username)

		w.WriteHeader(nethttp.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to login"})
		return
	}

	if !found || !user.IsActive || !authservice.CheckPassword(request.Password, user.PasswordHash) {
		w.WriteHeader(nethttp.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid username or password"})
		return
	}

	token, err := h.tokenManager.Generate(user)
	if err != nil {
		h.logger.Error("generate token failed", "error", err, "userId", user.ID)

		w.WriteHeader(nethttp.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to login"})
		return
	}

	_ = json.NewEncoder(w).Encode(loginResponse{
		Token: token,
		User:  toUserResponse(user),
	})
}

func (h *AuthHandler) Me(w nethttp.ResponseWriter, r *nethttp.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	claims, ok := CurrentUser(r.Context())
	if !ok {
		w.WriteHeader(nethttp.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	_ = json.NewEncoder(w).Encode(userResponse{
		ID:       claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	})
}

func toUserResponse(user domain.User) userResponse {
	return userResponse{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}
}
