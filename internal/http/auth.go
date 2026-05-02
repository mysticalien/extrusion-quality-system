package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	nethttp "net/http"
	"strings"

	authservice "extrusion-quality-system/internal/auth"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"
)

type contextKey string

const currentUserContextKey contextKey = "currentUser"

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
	Token string      `json:"token"`
	User  domain.User `json:"user"`
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
	if r.Method != nethttp.MethodPost {
		w.Header().Set("Allow", nethttp.MethodPost)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	defer r.Body.Close()

	var request loginRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid JSON body")
		return
	}

	user, found, err := h.userRepository.FindByUsername(r.Context(), strings.TrimSpace(request.Username))
	if err != nil {
		h.logger.Error("find user failed", "username", request.Username, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to login")
		return
	}

	if !found || !user.IsActive || !authservice.CheckPassword(request.Password, user.PasswordHash) {
		writeError(w, nethttp.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := h.tokenManager.Generate(user)
	if err != nil {
		h.logger.Error("generate token failed", "username", user.Username, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to generate token")
		return
	}

	writeJSON(w, nethttp.StatusOK, loginResponse{
		Token: token,
		User:  user,
	})
}

func (h *AuthHandler) Me(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		w.Header().Set("Allow", nethttp.MethodGet)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, ok := CurrentUser(r.Context())
	if !ok {
		writeError(w, nethttp.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, nethttp.StatusOK, user)
}

func AuthMiddleware(
	logger *slog.Logger,
	tokenManager *authservice.TokenManager,
	userRepository storage.UserRepository,
) func(nethttp.Handler) nethttp.Handler {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			rawToken, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				writeError(w, nethttp.StatusUnauthorized, "missing authorization token")
				return
			}

			claims, err := tokenManager.Parse(rawToken)
			if err != nil {
				status := nethttp.StatusUnauthorized
				message := "invalid authorization token"

				if errors.Is(err, authservice.ErrExpiredToken) {
					message = "authorization token expired"
				}

				writeError(w, status, message)
				return
			}

			user, found, err := userRepository.FindByID(r.Context(), claims.UserID)
			if err != nil {
				logger.Error("load current user failed", "userId", claims.UserID, "error", err)
				writeError(w, nethttp.StatusInternalServerError, "failed to load current user")
				return
			}

			if !found || !user.IsActive {
				writeError(w, nethttp.StatusUnauthorized, "user is inactive or not found")
				return
			}

			ctx := context.WithValue(r.Context(), currentUserContextKey, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CurrentUser(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(currentUserContextKey).(domain.User)

	return user, ok
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "

	if !strings.HasPrefix(header, prefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))

	return token, token != ""
}

func (h *AuthHandler) ChangePassword(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPost {
		w.Header().Set("Allow", nethttp.MethodPost)
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
		return
	}

	currentUser, ok := CurrentUser(r.Context())
	if !ok {
		writeError(w, nethttp.StatusUnauthorized, "unauthorized")
		return
	}

	defer r.Body.Close()

	var request domain.UserChangePassword

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := domain.ValidateUserChangePassword(request); err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	user, found, err := h.userRepository.FindByID(r.Context(), currentUser.ID)
	if err != nil {
		h.logger.Error("load current user failed", "userId", currentUser.ID, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load current user")
		return
	}

	if !found || !user.IsActive {
		writeError(w, nethttp.StatusUnauthorized, "user is inactive or not found")
		return
	}

	if !authservice.CheckPassword(request.OldPassword, user.PasswordHash) {
		writeError(w, nethttp.StatusUnauthorized, "old password is incorrect")
		return
	}

	newPasswordHash, err := authservice.HashPassword(request.NewPassword)
	if err != nil {
		h.logger.Error("hash new password failed", "userId", currentUser.ID, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to change password")
		return
	}

	updatedUser, found, err := h.userRepository.UpdatePassword(
		r.Context(),
		currentUser.ID,
		newPasswordHash,
	)
	if err != nil {
		h.logger.Error("change password failed", "userId", currentUser.ID, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to change password")
		return
	}

	if !found {
		writeError(w, nethttp.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, nethttp.StatusOK, updatedUser)
}
