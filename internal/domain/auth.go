package domain

type AuthClaims struct {
	UserID   UserID
	Username string
	Role     UserRole
}
