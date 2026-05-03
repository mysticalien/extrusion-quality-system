package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"extrusion-quality-system/internal/ports"
	"extrusion-quality-system/internal/security/token"
	"extrusion-quality-system/internal/usecase/auth"
	"log/slog"
	nethttp "net/http"
	"strings"

	"extrusion-quality-system/internal/domain"
)

type contextKey string

const currentUserContextKey contextKey = "currentUser"

type AuthHandler struct {
	logger       *slog.Logger
	authService  *auth.Service
	tokenManager ports.TokenManager
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
	authService *auth.Service,
	tokenManager ports.TokenManager,
) *AuthHandler {
	return &AuthHandler{
		logger:       logger,
		authService:  authService,
		tokenManager: tokenManager,
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

	h.logger.Debug("login attempt", "username", request.Username)

	result, err := h.authService.Login(r.Context(), auth.LoginInput{
		Username: request.Username,
		Password: request.Password,
	})
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			h.logger.Warn(
				"login failed",
				"username", request.Username,
				"reason", "invalid credentials",
			)

			writeError(w, nethttp.StatusUnauthorized, "invalid username or password")
			return
		}

		h.logger.Error("login failed", "username", request.Username, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to login")
		return
	}

	h.logger.Info(
		"login succeeded",
		"userId", result.User.ID,
		"username", result.User.Username,
		"role", result.User.Role,
	)

	writeJSON(w, nethttp.StatusOK, result)
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
	tokenManager ports.TokenManager,
	userRepository ports.UserRepository,
) func(nethttp.Handler) nethttp.Handler {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			rawToken, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				logger.Warn(
					"authorization failed",
					"reason", "missing authorization token",
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeError(w, nethttp.StatusUnauthorized, "missing authorization token")
				return
			}

			claims, err := tokenManager.Parse(rawToken)
			if err != nil {
				message := "invalid authorization token"

				if errors.Is(err, token.ErrExpiredToken) {
					message = "authorization token expired"
				}

				logger.Warn(
					"authorization failed",
					"reason", message,
					"method", r.Method,
					"path", r.URL.Path,
				)

				writeError(w, nethttp.StatusUnauthorized, message)
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

			logger.Debug(
				"request authorized",
				"userId", user.ID,
				"username", user.Username,
				"role", user.Role,
				"method", r.Method,
				"path", r.URL.Path,
			)

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

	updatedUser, err := h.authService.ChangePassword(
		r.Context(),
		auth.ChangePasswordInput{
			UserID:      currentUser.ID,
			OldPassword: request.OldPassword,
			NewPassword: request.NewPassword,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrOldPasswordIncorrect):
			writeError(w, nethttp.StatusUnauthorized, "old password is incorrect")
			return

		case errors.Is(err, auth.ErrNewPasswordSameAsOld):
			writeError(w, nethttp.StatusBadRequest, "new password must be different from old password")
			return

		case errors.Is(err, auth.ErrUserInactiveOrNotFound):
			writeError(w, nethttp.StatusUnauthorized, "user is inactive or not found")
			return

		default:
			h.logger.Error("change password failed", "userId", currentUser.ID, "error", err)
			writeError(w, nethttp.StatusInternalServerError, "failed to change password")
			return
		}
	}

	writeJSON(w, nethttp.StatusOK, updatedUser)
}
