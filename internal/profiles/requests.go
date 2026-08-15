package profiles

import (
	"strings"

	"github.com/MaksMakarskyi/booksy-go-api/internal/profiles/roles"
	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
)

var (
	_ valutils.Normalizer = (*CreateProfileReq)(nil)
)

type CreateProfileReq struct {
	Email    string     `json:"email"     validate:"required,max=255,email,endswith=@booksy.com"`
	Password string     `json:"password"  validate:"required,min=8,maxbytes=72,password"`
	FullName string     `json:"full_name" validate:"required,min=2,max=255"`
	Role     roles.Role `json:"role"      validate:"required,oneof=employee admin"`
}

func (cpr *CreateProfileReq) Normalize() {
	cpr.Email = strings.ToLower(strings.TrimSpace(cpr.Email))
	cpr.FullName = strings.TrimSpace(cpr.FullName)
}
