package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"extrusion-quality-system/internal/domain"
)

func TestServiceLoginSuccess(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}
	tokenManager := &fakeTokenManager{
		token: "test-token",
	}

	rawPassword := "correct-password"
	passwordHash, err := passwordHasher.Hash(rawPassword)
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: passwordHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	service := NewService(userRepository, passwordHasher, tokenManager)

	result, err := service.Login(context.Background(), LoginInput{
		Username: "maria.sokolova",
		Password: rawPassword,
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if result.Token != "test-token" {
		t.Fatalf("Token = %q, want %q", result.Token, "test-token")
	}

	if result.User.ID != 1 {
		t.Fatalf("User.ID = %d, want 1", result.User.ID)
	}

	if !tokenManager.generateCalled {
		t.Fatal("expected token manager Generate to be called")
	}
}

func TestServiceLoginWithWrongPasswordReturnsInvalidCredentials(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}
	tokenManager := &fakeTokenManager{
		token: "test-token",
	}

	passwordHash, err := passwordHasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: passwordHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	service := NewService(userRepository, passwordHasher, tokenManager)

	_, err = service.Login(context.Background(), LoginInput{
		Username: "maria.sokolova",
		Password: "wrong-password",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}

	if tokenManager.generateCalled {
		t.Fatal("token should not be generated for wrong password")
	}
}

func TestServiceLoginUnknownUserReturnsInvalidCredentials(t *testing.T) {
	service := NewService(
		newFakeUserRepository(),
		&fakePasswordHasher{},
		&fakeTokenManager{token: "test-token"},
	)

	_, err := service.Login(context.Background(), LoginInput{
		Username: "missing.user",
		Password: "password",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestServiceLoginInactiveUserReturnsInvalidCredentials(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}

	passwordHash, err := passwordHasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: passwordHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     false,
	}

	service := NewService(
		userRepository,
		passwordHasher,
		&fakeTokenManager{token: "test-token"},
	)

	_, err = service.Login(context.Background(), LoginInput{
		Username: "maria.sokolova",
		Password: "correct-password",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestServiceLoginRepositoryErrorIsReturned(t *testing.T) {
	userRepository := newFakeUserRepository()
	userRepository.findByUsernameErr = errUserRepositoryFailure

	service := NewService(
		userRepository,
		&fakePasswordHasher{},
		&fakeTokenManager{token: "test-token"},
	)

	_, err := service.Login(context.Background(), LoginInput{
		Username: "maria.sokolova",
		Password: "password",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestServiceChangePasswordSuccess(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}

	oldHash, err := passwordHasher.Hash("old-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: oldHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	service := NewService(
		userRepository,
		passwordHasher,
		&fakeTokenManager{token: "test-token"},
	)

	updatedUser, err := service.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:      1,
		OldPassword: "old-password",
		NewPassword: "new-password",
	})
	if err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}

	if updatedUser.ID != 1 {
		t.Fatalf("updated user id = %d, want 1", updatedUser.ID)
	}

	if !passwordHasher.Check("new-password", updatedUser.PasswordHash) {
		t.Fatal("new password hash does not match new password")
	}

	if passwordHasher.Check("old-password", updatedUser.PasswordHash) {
		t.Fatal("new password hash should not match old password")
	}
}

func TestServiceChangePasswordWrongOldPassword(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}

	oldHash, err := passwordHasher.Hash("old-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: oldHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	service := NewService(
		userRepository,
		passwordHasher,
		&fakeTokenManager{token: "test-token"},
	)

	_, err = service.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:      1,
		OldPassword: "wrong-password",
		NewPassword: "new-password",
	})

	if !errors.Is(err, ErrOldPasswordIncorrect) {
		t.Fatalf("error = %v, want ErrOldPasswordIncorrect", err)
	}
}

func TestServiceChangePasswordSameAsOldPassword(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}

	oldHash, err := passwordHasher.Hash("same-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: oldHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	service := NewService(
		userRepository,
		passwordHasher,
		&fakeTokenManager{token: "test-token"},
	)

	_, err = service.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:      1,
		OldPassword: "same-password",
		NewPassword: "same-password",
	})

	if !errors.Is(err, ErrNewPasswordSameAsOld) {
		t.Fatalf("error = %v, want ErrNewPasswordSameAsOld", err)
	}
}

func TestServiceChangePasswordMissingUser(t *testing.T) {
	service := NewService(
		newFakeUserRepository(),
		&fakePasswordHasher{},
		&fakeTokenManager{token: "test-token"},
	)

	_, err := service.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:      999,
		OldPassword: "old-password",
		NewPassword: "new-password",
	})

	if !errors.Is(err, ErrUserInactiveOrNotFound) {
		t.Fatalf("error = %v, want ErrUserInactiveOrNotFound", err)
	}
}

func TestServiceChangePasswordInactiveUser(t *testing.T) {
	userRepository := newFakeUserRepository()

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: "hash",
		Role:         domain.UserRoleTechnologist,
		IsActive:     false,
	}

	service := NewService(
		userRepository,
		&fakePasswordHasher{},
		&fakeTokenManager{token: "test-token"},
	)

	_, err := service.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:      1,
		OldPassword: "old-password",
		NewPassword: "new-password",
	})

	if !errors.Is(err, ErrUserInactiveOrNotFound) {
		t.Fatalf("error = %v, want ErrUserInactiveOrNotFound", err)
	}
}

type fakeUserRepository struct {
	users map[domain.UserID]domain.User

	findByUsernameErr error
	findByIDErr       error
	updatePasswordErr error

	updatePasswordFound    bool
	useUpdatePasswordFound bool
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		users: make(map[domain.UserID]domain.User),
	}
}

func (r *fakeUserRepository) All(ctx context.Context) ([]domain.User, error) {
	_ = ctx

	users := make([]domain.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	return users, nil
}

func (r *fakeUserRepository) FindByUsername(
	ctx context.Context,
	username string,
) (domain.User, bool, error) {
	_ = ctx

	if r.findByUsernameErr != nil {
		return domain.User{}, false, r.findByUsernameErr
	}

	for _, user := range r.users {
		if user.Username == username {
			return user, true, nil
		}
	}

	return domain.User{}, false, nil
}

func (r *fakeUserRepository) FindByID(
	ctx context.Context,
	id domain.UserID,
) (domain.User, bool, error) {
	_ = ctx

	if r.findByIDErr != nil {
		return domain.User{}, false, r.findByIDErr
	}

	user, found := r.users[id]

	return user, found, nil
}

func (r *fakeUserRepository) Create(
	ctx context.Context,
	user domain.User,
) (domain.User, error) {
	_ = ctx

	if user.ID == 0 {
		user.ID = domain.UserID(len(r.users) + 1)
	}

	r.users[user.ID] = user

	return user, nil
}

func (r *fakeUserRepository) UpdateRole(
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
	user.UpdatedAt = time.Now().UTC()
	r.users[id] = user

	return user, true, nil
}

func (r *fakeUserRepository) UpdatePassword(
	ctx context.Context,
	id domain.UserID,
	passwordHash string,
) (domain.User, bool, error) {
	_ = ctx

	if r.updatePasswordErr != nil {
		return domain.User{}, false, r.updatePasswordErr
	}

	if r.useUpdatePasswordFound && !r.updatePasswordFound {
		return domain.User{}, false, nil
	}

	user, found := r.users[id]
	if !found {
		return domain.User{}, false, nil
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now().UTC()
	r.users[id] = user

	return user, true, nil
}

func (r *fakeUserRepository) SetActive(
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
	user.UpdatedAt = time.Now().UTC()
	r.users[id] = user

	return user, true, nil
}

type fakePasswordHasher struct {
	hashErr          error
	checkOK          bool
	forceCheckResult bool
}

func (h *fakePasswordHasher) Hash(password string) (string, error) {
	if h.hashErr != nil {
		return "", h.hashErr
	}

	return "hashed:" + password, nil
}

func (h *fakePasswordHasher) Check(password string, passwordHash string) bool {
	if h.forceCheckResult {
		return h.checkOK
	}

	return passwordHash == "hashed:"+password
}

type fakeTokenManager struct {
	token          string
	generateErr    error
	parseErr       error
	generateCalled bool
}

func (m *fakeTokenManager) Generate(user domain.User) (string, error) {
	m.generateCalled = true

	if m.generateErr != nil {
		return "", m.generateErr
	}

	return m.token, nil
}

func (m *fakeTokenManager) Parse(rawToken string) (domain.AuthClaims, error) {
	if m.parseErr != nil {
		return domain.AuthClaims{}, m.parseErr
	}

	return domain.AuthClaims{
		UserID:   1,
		Username: "maria.sokolova",
		Role:     domain.UserRoleTechnologist,
	}, nil
}

func TestServiceLoginTrimsUsername(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}
	tokenManager := &fakeTokenManager{
		token: "test-token",
	}

	passwordHash, err := passwordHasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: passwordHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	service := NewService(userRepository, passwordHasher, tokenManager)

	result, err := service.Login(context.Background(), LoginInput{
		Username: "  maria.sokolova  ",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	if result.User.Username != "maria.sokolova" {
		t.Fatalf("username = %q, want maria.sokolova", result.User.Username)
	}
}

func TestServiceLoginTokenGenerationErrorIsReturned(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}

	passwordHash, err := passwordHasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: passwordHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	tokenManager := &fakeTokenManager{
		generateErr: errTokenGenerationFailure,
	}

	service := NewService(userRepository, passwordHasher, tokenManager)

	_, err = service.Login(context.Background(), LoginInput{
		Username: "maria.sokolova",
		Password: "correct-password",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errTokenGenerationFailure) {
		t.Fatalf("error = %v, want errTokenGenerationFailure", err)
	}
}

func TestServiceChangePasswordHashErrorIsReturned(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}

	oldHash, err := passwordHasher.Hash("old-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: oldHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	service := NewService(
		userRepository,
		&fakePasswordHasher{
			hashErr: errPasswordHashFailure,
		},
		&fakeTokenManager{token: "test-token"},
	)

	_, err = service.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:      1,
		OldPassword: "old-password",
		NewPassword: "new-password",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errPasswordHashFailure) {
		t.Fatalf("error = %v, want errPasswordHashFailure", err)
	}
}

func TestServiceChangePasswordUpdateRepositoryErrorIsReturned(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}

	oldHash, err := passwordHasher.Hash("old-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: oldHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	userRepository.updatePasswordErr = errUserRepositoryFailure

	service := NewService(
		userRepository,
		passwordHasher,
		&fakeTokenManager{token: "test-token"},
	)

	_, err = service.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:      1,
		OldPassword: "old-password",
		NewPassword: "new-password",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errUserRepositoryFailure) {
		t.Fatalf("error = %v, want errUserRepositoryFailure", err)
	}
}

func TestServiceLoginPasswordHasherRejectsPassword(t *testing.T) {
	userRepository := newFakeUserRepository()

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: "any-hash",
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	passwordHasher := &fakePasswordHasher{
		forceCheckResult: true,
		checkOK:          false,
	}

	tokenManager := &fakeTokenManager{
		token: "test-token",
	}

	service := NewService(userRepository, passwordHasher, tokenManager)

	_, err := service.Login(context.Background(), LoginInput{
		Username: "maria.sokolova",
		Password: "correct-password",
	})

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}

	if tokenManager.generateCalled {
		t.Fatal("token should not be generated")
	}
}

func TestServiceChangePasswordFindByIDErrorIsReturned(t *testing.T) {
	userRepository := newFakeUserRepository()
	userRepository.findByIDErr = errUserRepositoryFailure

	service := NewService(
		userRepository,
		&fakePasswordHasher{},
		&fakeTokenManager{token: "test-token"},
	)

	_, err := service.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:      1,
		OldPassword: "old-password",
		NewPassword: "new-password",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, errUserRepositoryFailure) {
		t.Fatalf("error = %v, want errUserRepositoryFailure", err)
	}
}

func TestServiceChangePasswordUpdateReturnsNotFound(t *testing.T) {
	userRepository := newFakeUserRepository()
	passwordHasher := &fakePasswordHasher{}

	oldHash, err := passwordHasher.Hash("old-password")
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	userRepository.users[1] = domain.User{
		ID:           1,
		Username:     "maria.sokolova",
		PasswordHash: oldHash,
		Role:         domain.UserRoleTechnologist,
		IsActive:     true,
	}

	userRepository.useUpdatePasswordFound = true
	userRepository.updatePasswordFound = false

	service := NewService(
		userRepository,
		passwordHasher,
		&fakeTokenManager{token: "test-token"},
	)

	_, err = service.ChangePassword(context.Background(), ChangePasswordInput{
		UserID:      1,
		OldPassword: "old-password",
		NewPassword: "new-password",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrUserInactiveOrNotFound) {
		t.Fatalf("error = %v, want ErrUserInactiveOrNotFound", err)
	}
}

var errUserRepositoryFailure = errors.New("user repository failure")
var errTokenGenerationFailure = errors.New("token generation failure")
var errPasswordHashFailure = errors.New("password hash failure")
