package auth

import (
	"strings"

	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
)

var _ valutils.Normalizer = (*CreateTokenReq)(nil)

type CreateTokenReq struct {
	Email    string `json:"email"    validate:"required,max=255,email"`
	Password string `json:"password" validate:"required,min=8"`
}

func (ctr *CreateTokenReq) Normalize() {
	ctr.Email = strings.ToLower(strings.TrimSpace(ctr.Email))
}
