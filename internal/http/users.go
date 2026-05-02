package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	nethttp "net/http"
	"strconv"
	"strings"

	authservice "extrusion-quality-system/internal/auth"
	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/storage"

	"github.com/jackc/pgx/v5/pgconn"
)

type UserHandler struct {
	logger         *slog.Logger
	userRepository storage.UserRepository
}

func NewUserHandler(
	logger *slog.Logger,
	userRepository storage.UserRepository,
) *UserHandler {
	return &UserHandler{
		logger:         logger,
		userRepository: userRepository,
	}
}

func (h *UserHandler) ListCreate(w nethttp.ResponseWriter, r *nethttp.Request) {
	switch r.Method {
	case nethttp.MethodGet:
		h.List(w, r)

	case nethttp.MethodPost:
		h.Create(w, r)

	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *UserHandler) List(w nethttp.ResponseWriter, r *nethttp.Request) {
	users, err := h.userRepository.All(r.Context())
	if err != nil {
		h.logger.Error("load users failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to load users")
		return
	}

	writeJSON(w, nethttp.StatusOK, users)
}

func (h *UserHandler) Create(w nethttp.ResponseWriter, r *nethttp.Request) {
	defer r.Body.Close()

	var request domain.UserCreate

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := domain.ValidateUserCreate(request); err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	passwordHash, err := authservice.HashPassword(request.Password)
	if err != nil {
		h.logger.Error("hash password failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to create user")
		return
	}

	user := domain.User{
		Username:     strings.TrimSpace(request.Username),
		PasswordHash: passwordHash,
		Role:         request.Role,
		IsActive:     request.IsActive,
	}

	created, err := h.userRepository.Create(r.Context(), user)
	if err != nil {
		if isUniqueViolation(err) || errors.Is(err, storage.ErrMemoryUserAlreadyExists) {
			writeError(w, nethttp.StatusConflict, "user already exists")
			return
		}

		h.logger.Error("create user failed", "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to create user")
		return
	}

	h.logger.Info(
		"user created",
		"userId", user.ID,
		"username", user.Username,
		"role", user.Role,
		"isActive", user.IsActive,
	)

	writeJSON(w, nethttp.StatusCreated, created)
}

func (h *UserHandler) Action(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, action, ok := parseUserActionPath(r.URL.Path)
	if !ok {
		writeError(w, nethttp.StatusNotFound, "not found")
		return
	}

	switch action {
	case "role":
		if r.Method != nethttp.MethodPatch {
			w.Header().Set("Allow", nethttp.MethodPatch)
			writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
			return
		}

		h.UpdateRole(w, r, id)

	case "activate":
		if r.Method != nethttp.MethodPost {
			w.Header().Set("Allow", nethttp.MethodPost)
			writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
			return
		}

		h.SetActive(w, r, id, true)

	case "deactivate":
		if r.Method != nethttp.MethodPost {
			w.Header().Set("Allow", nethttp.MethodPost)
			writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
			return
		}

		h.SetActive(w, r, id, false)

	case "reset-password":
		if r.Method != nethttp.MethodPost {
			w.Header().Set("Allow", nethttp.MethodPost)
			writeError(w, nethttp.StatusMethodNotAllowed, "method not allowed")
			return
		}

		h.ResetPassword(w, r, id)

	default:
		writeError(w, nethttp.StatusNotFound, "not found")
	}
}

func (h *UserHandler) UpdateRole(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
	id domain.UserID,
) {
	defer r.Body.Close()

	var request domain.UserRoleUpdate

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := domain.ValidateUserRoleUpdate(request); err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	user, found, err := h.userRepository.UpdateRole(r.Context(), id, request.Role)
	if err != nil {
		h.logger.Error("update user role failed", "id", id, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to update user role")
		return
	}

	if !found {
		writeError(w, nethttp.StatusNotFound, "user not found")
		return
	}

	h.logger.Info(
		"user role changed",
		"userId", user.ID,
		"username", user.Username,
		"role", user.Role,
	)

	writeJSON(w, nethttp.StatusOK, user)
}

func (h *UserHandler) SetActive(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
	id domain.UserID,
	isActive bool,
) {
	user, found, err := h.userRepository.SetActive(r.Context(), id, isActive)
	if err != nil {
		h.logger.Error("update user active status failed", "id", id, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to update user")
		return
	}

	if !found {
		writeError(w, nethttp.StatusNotFound, "user not found")
		return
	}

	h.logger.Info(
		"user activation changed",
		"userId", user.ID,
		"username", user.Username,
		"isActive", user.IsActive,
	)

	writeJSON(w, nethttp.StatusOK, user)
}

func (h *UserHandler) ResetPassword(
	w nethttp.ResponseWriter,
	r *nethttp.Request,
	id domain.UserID,
) {
	defer r.Body.Close()

	var request domain.UserPasswordUpdate

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := domain.ValidateUserPasswordUpdate(request); err != nil {
		writeError(w, nethttp.StatusBadRequest, err.Error())
		return
	}

	passwordHash, err := authservice.HashPassword(request.Password)
	if err != nil {
		h.logger.Error("hash password failed", "id", id, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to reset password")
		return
	}

	user, found, err := h.userRepository.UpdatePassword(r.Context(), id, passwordHash)
	if err != nil {
		h.logger.Error("reset password failed", "id", id, "error", err)
		writeError(w, nethttp.StatusInternalServerError, "failed to reset password")
		return
	}

	if !found {
		writeError(w, nethttp.StatusNotFound, "user not found")
		return
	}

	h.logger.Info(
		"user password reset",
		"userId", user.ID,
		"username", user.Username,
	)

	writeJSON(w, nethttp.StatusOK, user)
}

func parseUserActionPath(path string) (domain.UserID, string, bool) {
	const prefix = "/api/users/"

	if !strings.HasPrefix(path, prefix) {
		return 0, "", false
	}

	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(rest, "/")

	if len(parts) != 2 {
		return 0, "", false
	}

	rawID := parts[0]
	action := parts[1]

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}

	return domain.UserID(id), action, true
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
