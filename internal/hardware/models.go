package hardware

import (
	"strings"

	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
)

type Hardware struct {
	ID           int            `json:"id"`
	Name         string         `json:"name"`
	Brand        string         `json:"brand"`
	Description  *string        `json:"description"`
	PurchaseDate *string        `json:"purchase_date"`
	Status       HardwareStatus `json:"status"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
}

type HardwareStatus string

const (
	Available HardwareStatus = "available"
	InUse     HardwareStatus = "in_use"
	Repair    HardwareStatus = "repair"
)

var _ valutils.Normalizer = (*NewHardware)(nil)

type NewHardware struct {
	Name         string  `json:"name"          validate:"required,min=1,max=255"`
	Brand        string  `json:"brand"         validate:"required,min=1,max=255"`
	Description  *string `json:"description"   validate:"omitempty,min=1,max=4096"`
	PurchaseDate *string `json:"purchase_date" validate:"omitempty,datetime=2006-01-02,notfuture"`
}

func (nh *NewHardware) Normalize() {
	nh.Name = strings.TrimSpace(nh.Name)
	nh.Brand = strings.TrimSpace(nh.Brand)
	nh.Description = trimmedOrNil(nh.Description)
	nh.PurchaseDate = trimmedOrNil(nh.PurchaseDate)
}

var (
	_ valutils.Normalizer    = (*UpdatedHardware)(nil)
	_ valutils.SelfValidator = (*UpdatedHardware)(nil)
)

type UpdatedHardware struct {
	ID           int     `param:"id"`
	Name         *string `json:"name"          validate:"omitempty,min=1,max=255"`
	Brand        *string `json:"brand"         validate:"omitempty,min=1,max=255"`
	Description  *string `json:"description"   validate:"omitempty,min=1,max=4096"`
	PurchaseDate *string `json:"purchase_date" validate:"omitempty,datetime=2006-01-02,notfuture"`
}

func (uh *UpdatedHardware) Normalize() {
	uh.Name = trimmedOrNil(uh.Name)
	uh.Brand = trimmedOrNil(uh.Brand)
	uh.Description = trimmedOrNil(uh.Description)
	uh.PurchaseDate = trimmedOrNil(uh.PurchaseDate)
}

func (uh *UpdatedHardware) SelfValidate() []valutils.FieldError {
	if uh.Name == nil && uh.Brand == nil && uh.Description == nil && uh.PurchaseDate == nil {
		return []valutils.FieldError{{
			Rule:    "no_updates",
			Message: "provide at least one field to update",
		}}
	}

	return nil
}

func trimmedOrNil(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
