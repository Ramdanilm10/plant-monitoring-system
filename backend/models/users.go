package models

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

type LoginResponse struct {
	Token            string `json:"token"`
	TokenType        string `json:"token_type"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
	User             User   `json:"user"`
}
