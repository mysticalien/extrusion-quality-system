package auth

import "errors"

var (
	ErrInvalidCredentials     = errors.New("invalid username or password")
	ErrUserInactiveOrNotFound = errors.New("user is inactive or not found")
	ErrOldPasswordIncorrect   = errors.New("old password is incorrect")
	ErrNewPasswordSameAsOld   = errors.New("new password must be different from old password")
)
