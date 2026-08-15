package auth

import "github.com/MaksMakarskyi/booksy-go-api/internal/profiles/roles"

type User struct {
	ID       int        `json:"id"`
	Email    string     `json:"email"`
	FullName string     `json:"full_name"`
	Role     roles.Role `json:"role"`
}

type UserWithCreds struct {
	User
	PasswordHash string `db:"password_hash" json:"-"`
}
