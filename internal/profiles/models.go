package profiles

import "github.com/MaksMakarskyi/booksy-go-api/internal/profiles/roles"

type Profile struct {
	ID        int        `json:"id"`
	Email     string     `json:"email"`
	FullName  string     `json:"full_name"`
	Role      roles.Role `json:"role"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
}

type NewProfile struct {
	Email        string
	FullName     string
	Role         roles.Role
	PasswordHash string
}
