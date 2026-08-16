package hardware

import (
	"strings"

	valutils "github.com/MaksMakarskyi/booksy-go-api/internal/utils/validation"
)

type Hardware struct {
	ID           int            `json:"id"`
	Name         string         `json:"name"`
	Brand        string         `json:"brand"`
	Description  string         `json:"description"`
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

func (s HardwareStatus) IsValid() bool {
	switch s {
	case Available, InUse, Repair:
		return true
	default:
		return false
	}
}

var _ valutils.Normalizer = (*NewHardware)(nil)

type NewHardware struct {
	Name         string `json:"name"          validate:"required,min=1,max=255"`
	Brand        string `json:"brand"         validate:"required,min=1,max=255"`
	Description  string `json:"description"   validate:"omitempty,max=4096"`
	PurchaseDate string `json:"purchase_date" validate:"omitempty,date,notfuture"`
}

func (nh *NewHardware) Normalize() {
	nh.Name = strings.TrimSpace(nh.Name)
	nh.Brand = strings.TrimSpace(nh.Brand)
	nh.Description = strings.TrimSpace(nh.Description)
	nh.PurchaseDate = strings.TrimSpace(nh.PurchaseDate)
}

var (
	_ valutils.Normalizer    = (*UpdatedHardware)(nil)
	_ valutils.SelfValidator = (*UpdatedHardware)(nil)
)

type UpdatedHardware struct {
	ID           int     `json:"-"             param:"id"`
	Name         *string `json:"name"          validate:"omitempty,min=1,max=255"`
	Brand        *string `json:"brand"         validate:"omitempty,min=1,max=255"`
	Description  *string `json:"description"   validate:"omitempty,max=4096"`
	PurchaseDate *string `json:"purchase_date" validate:"omitempty,date,notfuture"`
}

func (uh *UpdatedHardware) Normalize() {
	uh.Name = trimmed(uh.Name)
	uh.Brand = trimmed(uh.Brand)
	uh.Description = trimmed(uh.Description)
	uh.PurchaseDate = trimmed(uh.PurchaseDate)
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

func trimmed(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)

	return &trimmed
}
