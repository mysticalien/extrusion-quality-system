package auth

import (
	"context"
	"fmt"
	"strings"

	"extrusion-quality-system/internal/domain"
	"extrusion-quality-system/internal/ports"
)

type Service struct {
	userRepository ports.UserRepository
	passwordHasher ports.PasswordHasher
	tokenManager   ports.TokenManager
}

func NewService(
	userRepository ports.UserRepository,
	passwordHasher ports.PasswordHasher,
	tokenManager ports.TokenManager,
) *Service {
	return &Service{
		userRepository: userRepository,
		passwordHasher: passwordHasher,
		tokenManager:   tokenManager,
	}
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	username := strings.TrimSpace(input.Username)

	user, found, err := s.userRepository.FindByUsername(ctx, username)
	if err != nil {
		return LoginResult{}, fmt.Errorf("find user by username: %w", err)
	}

	if !found || !user.IsActive {
		return LoginResult{}, ErrInvalidCredentials
	}

	if !s.passwordHasher.Check(input.Password, user.PasswordHash) {
		return LoginResult{}, ErrInvalidCredentials
	}

	rawToken, err := s.tokenManager.Generate(user)
	if err != nil {
		return LoginResult{}, fmt.Errorf("generate token: %w", err)
	}

	return LoginResult{
		Token: rawToken,
		User:  user,
	}, nil
}

func (s *Service) CurrentUser(ctx context.Context, userID domain.UserID) (domain.User, error) {
	user, found, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("find user by id: %w", err)
	}

	if !found || !user.IsActive {
		return domain.User{}, ErrUserInactiveOrNotFound
	}

	return user, nil
}

func (s *Service) ChangePassword(
	ctx context.Context,
	input ChangePasswordInput,
) (domain.User, error) {
	user, found, err := s.userRepository.FindByID(ctx, input.UserID)
	if err != nil {
		return domain.User{}, fmt.Errorf("find user by id: %w", err)
	}

	if !found || !user.IsActive {
		return domain.User{}, ErrUserInactiveOrNotFound
	}

	if !s.passwordHasher.Check(input.OldPassword, user.PasswordHash) {
		return domain.User{}, ErrOldPasswordIncorrect
	}

	if s.passwordHasher.Check(input.NewPassword, user.PasswordHash) {
		return domain.User{}, ErrNewPasswordSameAsOld
	}

	newPasswordHash, err := s.passwordHasher.Hash(input.NewPassword)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash new password: %w", err)
	}

	updatedUser, found, err := s.userRepository.UpdatePassword(
		ctx,
		user.ID,
		newPasswordHash,
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("update password: %w", err)
	}

	if !found {
		return domain.User{}, ErrUserInactiveOrNotFound
	}

	return updatedUser, nil
}
