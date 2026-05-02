package auth

import "golang.org/x/crypto/bcrypt"

func CheckPassword(password string, passwordHash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))

	return err == nil
}
