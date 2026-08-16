package hardware

import (
	"strings"

	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
)

var _ valutils.Normalizer = (*SearchRequest)(nil)

type SearchRequest struct {
	Query string `query:"query" validate:"required,min=1,max=512"`
}

func (r *SearchRequest) Normalize() {
	r.Query = strings.TrimSpace(r.Query)
}
