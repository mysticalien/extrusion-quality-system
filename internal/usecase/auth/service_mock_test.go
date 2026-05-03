package auth

import (
	"context"
	"errors"
	"testing"

	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/mocks"

	"github.com/stretchr/testify/require"
)

func TestServiceLoginSuccessWithMocks(t *testing.T) {
	ctx := context.Background()

	user := domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: "hashed-password",
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	userRepository := mocks.NewUserRepositoryMock(t)
	passwordHasher := mocks.NewPasswordHasherMock(t)
	tokenManager := mocks.NewTokenManagerMock(t)

	userRepository.
		EXPECT().
		FindByUsername(ctx, "maria.sokolova").
		Return(user, true, nil).
		Once()

	passwordHasher.
		EXPECT().
		Check("correct-password", "hashed-password").
		Return(true).
		Once()

	tokenManager.
		EXPECT().
		Generate(user).
		Return("jwt-token", nil).
		Once()

	service := NewService(userRepository, passwordHasher, tokenManager)

	result, err := service.Login(ctx, LoginInput{
		Username: "maria.sokolova",
		Password: "correct-password",
	})

	require.NoError(t, err)
	require.Equal(t, "jwt-token", result.Token)
	require.Equal(t, user.ID, result.User.ID)
	require.Equal(t, user.Username, result.User.Username)
}

func TestServiceLoginWrongPasswordWithMocks(t *testing.T) {
	ctx := context.Background()

	user := domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: "hashed-password",
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	userRepository := mocks.NewUserRepositoryMock(t)
	passwordHasher := mocks.NewPasswordHasherMock(t)
	tokenManager := mocks.NewTokenManagerMock(t)

	userRepository.
		EXPECT().
		FindByUsername(ctx, "maria.sokolova").
		Return(user, true, nil).
		Once()

	passwordHasher.
		EXPECT().
		Check("wrong-password", "hashed-password").
		Return(false).
		Once()

	service := NewService(userRepository, passwordHasher, tokenManager)

	_, err := service.Login(ctx, LoginInput{
		Username: "maria.sokolova",
		Password: "wrong-password",
	})

	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestServiceLoginTokenGenerationErrorWithMocks(t *testing.T) {
	ctx := context.Background()

	tokenErr := errors.New("token generation failed")

	user := domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: "hashed-password",
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	userRepository := mocks.NewUserRepositoryMock(t)
	passwordHasher := mocks.NewPasswordHasherMock(t)
	tokenManager := mocks.NewTokenManagerMock(t)

	userRepository.
		EXPECT().
		FindByUsername(ctx, "maria.sokolova").
		Return(user, true, nil).
		Once()

	passwordHasher.
		EXPECT().
		Check("correct-password", "hashed-password").
		Return(true).
		Once()

	tokenManager.
		EXPECT().
		Generate(user).
		Return("", tokenErr).
		Once()

	service := NewService(userRepository, passwordHasher, tokenManager)

	_, err := service.Login(ctx, LoginInput{
		Username: "maria.sokolova",
		Password: "correct-password",
	})

	require.ErrorIs(t, err, tokenErr)
}

func TestServiceChangePasswordSuccessWithMocks(t *testing.T) {
	ctx := context.Background()

	user := domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: "old-hash",
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	updatedUser := user
	updatedUser.PasswordHash = "new-hash"

	userRepository := mocks.NewUserRepositoryMock(t)
	passwordHasher := mocks.NewPasswordHasherMock(t)
	tokenManager := mocks.NewTokenManagerMock(t)

	userRepository.
		EXPECT().
		FindByID(ctx, domain.UserID(1)).
		Return(user, true, nil).
		Once()

	passwordHasher.
		EXPECT().
		Check("old-password", "old-hash").
		Return(true).
		Once()

	passwordHasher.
		EXPECT().
		Check("new-password", "old-hash").
		Return(false).
		Once()

	passwordHasher.
		EXPECT().
		Hash("new-password").
		Return("new-hash", nil).
		Once()

	userRepository.
		EXPECT().
		UpdatePassword(ctx, domain.UserID(1), "new-hash").
		Return(updatedUser, true, nil).
		Once()

	service := NewService(userRepository, passwordHasher, tokenManager)

	result, err := service.ChangePassword(ctx, ChangePasswordInput{
		UserID:      1,
		OldPassword: "old-password",
		NewPassword: "new-password",
	})

	require.NoError(t, err)
	require.Equal(t, "new-hash", result.PasswordHash)
}
