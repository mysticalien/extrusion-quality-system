package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ports"
	authusecase "extrusion-quality-system/internal/usecase/auth"
)

func TestAuthHandlerLoginSuccess(t *testing.T) {
	userRepository := newHTTPFakeUserRepository()
	passwordHasher := &httpFakePasswordHasher{}
	tokenManager := &httpFakeTokenManager{
		token: "test-token",
	}

	hash, err := passwordHasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: hash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	authService := authusecase.NewService(
		userRepository,
		passwordHasher,
		tokenManager,
	)

	handler := NewAuthHandler(
		slog.Default(),
		authService,
		tokenManager,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(`{"username":"maria.sokolova","password":"correct-password"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	handler.Login(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var body loginResponse

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Token != "test-token" {
		t.Fatalf("token = %q, want test-token", body.Token)
	}

	if body.User.Username != "maria.sokolova" {
		t.Fatalf("username = %q, want maria.sokolova", body.User.Username)
	}
}

func TestAuthHandlerLoginInvalidPasswordReturnsUnauthorized(t *testing.T) {
	userRepository := newHTTPFakeUserRepository()
	passwordHasher := &httpFakePasswordHasher{}
	tokenManager := &httpFakeTokenManager{
		token: "test-token",
	}

	hash, err := passwordHasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: hash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	authService := authusecase.NewService(
		userRepository,
		passwordHasher,
		tokenManager,
	)

	handler := NewAuthHandler(
		slog.Default(),
		authService,
		tokenManager,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(`{"username":"maria.sokolova","password":"wrong-password"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	handler.Login(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	var body errorResponse

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.Error.Code != "invalid_username_or_password" {
		t.Fatalf("error code = %q, want invalid_username_or_password", body.Error.Code)
	}
}

func TestAuthHandlerLoginInvalidJSONReturnsBadRequest(t *testing.T) {
	handler := NewAuthHandler(
		slog.Default(),
		authusecase.NewService(
			newHTTPFakeUserRepository(),
			&httpFakePasswordHasher{},
			&httpFakeTokenManager{},
		),
		&httpFakeTokenManager{},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/login",
		strings.NewReader(`{"username":`),
	)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()

	handler.Login(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestAuthMiddlewareMissingTokenReturnsUnauthorized(t *testing.T) {
	middleware := AuthMiddleware(
		slog.Default(),
		&httpFakeTokenManager{},
		newHTTPFakeUserRepository(),
	)

	handlerCalled := false

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	if handlerCalled {
		t.Fatal("handler should not be called")
	}
}

func TestAuthMiddlewareValidTokenAddsCurrentUser(t *testing.T) {
	userRepository := newHTTPFakeUserRepository()

	userRepository.users[2] = domain.User{
		ID:       2,
		Username: "maria.sokolova",
		Role:     domain.UserRoleTechnologist,
		IsActive: true,
	}

	tokenManager := &httpFakeTokenManager{
		claims: domain.AuthClaims{
			UserID:   2,
			Username: "maria.sokolova",
			Role:     domain.UserRoleTechnologist,
		},
	}

	middleware := AuthMiddleware(
		slog.Default(),
		tokenManager,
		userRepository,
	)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := CurrentUser(r.Context())
		if !ok {
			t.Fatal("expected current user in context")
		}

		if user.ID != 2 {
			t.Fatalf("current user id = %d, want 2", user.ID)
		}

		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer valid-token")

	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

type httpFakeUserRepository struct {
	users map[domain.UserID]domain.User
}

func newHTTPFakeUserRepository() *httpFakeUserRepository {
	return &httpFakeUserRepository{
		users: make(map[domain.UserID]domain.User),
	}
}

func (r *httpFakeUserRepository) All(ctx context.Context) ([]domain.User, error) {
	_ = ctx

	users := make([]domain.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	return users, nil
}

func (r *httpFakeUserRepository) FindByUsername(
	ctx context.Context,
	username string,
) (domain.User, bool, error) {
	_ = ctx

	for _, user := range r.users {
		if user.Username == username {
			return user, true, nil
		}
	}

	return domain.User{}, false, nil
}

func (r *httpFakeUserRepository) FindByID(
	ctx context.Context,
	id domain.UserID,
) (domain.User, bool, error) {
	_ = ctx

	user, found := r.users[id]

	return user, found, nil
}

func (r *httpFakeUserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	_ = ctx

	if user.ID == 0 {
		user.ID = domain.UserID(len(r.users) + 1)
	}

	r.users[user.ID] = user

	return user, nil
}

func (r *httpFakeUserRepository) UpdateRole(
	ctx context.Context,
	id domain.UserID,
	role domain.UserRole,
) (domain.User, bool, error) {
	_ = ctx

	user, found := r.users[id]
	if !found {
		return domain.User{}, false, nil
	}

	user.Role = role
	r.users[id] = user

	return user, true, nil
}

func (r *httpFakeUserRepository) UpdatePassword(
	ctx context.Context,
	id domain.UserID,
	passwordHash string,
) (domain.User, bool, error) {
	_ = ctx

	user, found := r.users[id]
	if !found {
		return domain.User{}, false, nil
	}

	user.PasswordHash = passwordHash
	r.users[id] = user

	return user, true, nil
}

func (r *httpFakeUserRepository) SetActive(
	ctx context.Context,
	id domain.UserID,
	isActive bool,
) (domain.User, bool, error) {
	_ = ctx

	user, found := r.users[id]
	if !found {
		return domain.User{}, false, nil
	}

	user.IsActive = isActive
	r.users[id] = user

	return user, true, nil
}

type httpFakePasswordHasher struct{}

func (h *httpFakePasswordHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (h *httpFakePasswordHasher) Check(password string, passwordHash string) bool {
	return passwordHash == "hashed:"+password
}

type httpFakeTokenManager struct {
	token  string
	claims domain.AuthClaims
	err    error
}

func (m *httpFakeTokenManager) Generate(user domain.User) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	if m.token == "" {
		return "test-token", nil
	}

	return m.token, nil
}

func (m *httpFakeTokenManager) Parse(rawToken string) (domain.AuthClaims, error) {
	if m.err != nil {
		return domain.AuthClaims{}, m.err
	}

	if rawToken == "" {
		return domain.AuthClaims{}, errors.New("empty token")
	}

	if m.claims.UserID == 0 {
		return domain.AuthClaims{
			UserID:   1,
			Username: "test",
			Role:     domain.UserRoleOperator,
		}, nil
	}

	return m.claims, nil
}

var _ ports.UserRepository = (*httpFakeUserRepository)(nil)
var _ ports.PasswordHasher = (*httpFakePasswordHasher)(nil)
var _ ports.TokenManager = (*httpFakeTokenManager)(nil)
