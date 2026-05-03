package httpadapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"extrusion-quality-system/internal/domain"
)

func TestRequireRolesOperatorForbidden(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/setpoints", nil)
	request = request.WithContext(context.WithValue(
		request.Context(),
		currentUserContextKey,
		domain.User{
			ID:       1,
			Username: "ivan.petrov",
			Role:     domain.UserRoleOperator,
			IsActive: true,
		},
	))

	response := httptest.NewRecorder()

	handlerCalled := false

	handler := RequireRoles(domain.UserRoleTechnologist)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}),
	)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}

	if handlerCalled {
		t.Fatal("handler should not be called")
	}
}

func TestRequireRolesTechnologistAllowed(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/setpoints", nil)
	request = request.WithContext(context.WithValue(
		request.Context(),
		currentUserContextKey,
		domain.User{
			ID:       2,
			Username: "maria.sokolova",
			Role:     domain.UserRoleTechnologist,
			IsActive: true,
		},
	))

	response := httptest.NewRecorder()

	handlerCalled := false

	handler := RequireRoles(domain.UserRoleTechnologist)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}),
	)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if !handlerCalled {
		t.Fatal("handler should be called")
	}
}

func TestRequireRolesAdminAllowedWhenAdminIsAllowed(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/setpoints", nil)
	request = request.WithContext(context.WithValue(
		request.Context(),
		currentUserContextKey,
		domain.User{
			ID:       3,
			Username: "admin.local",
			Role:     domain.UserRoleAdmin,
			IsActive: true,
		},
	))

	response := httptest.NewRecorder()

	handlerCalled := false

	handler := RequireRoles(domain.UserRoleTechnologist, domain.UserRoleAdmin)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}),
	)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if !handlerCalled {
		t.Fatal("handler should be called")
	}
}

func TestRequireRolesWithoutCurrentUserReturnsUnauthorized(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/setpoints", nil)
	response := httptest.NewRecorder()

	handlerCalled := false

	handler := RequireRoles(domain.UserRoleTechnologist)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}),
	)

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	if handlerCalled {
		t.Fatal("handler should not be called")
	}
}
