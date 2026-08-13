package hardware

import "strings"

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

type NewHardware struct {
	Name         string  `json:"name"          validate:"required,min=2,max=255"`
	Brand        string  `json:"brand"         validate:"required,min=2,max=255"`
	Description  *string `json:"description"   validate:"omitempty,max=4096"`
	PurchaseDate *string `json:"purchase_date" validate:"omitempty,datetime=2006-01-02,notfuture"`
}

func (nh *NewHardware) Normalize() {
	nh.Name = strings.TrimSpace(nh.Name)
	nh.Brand = strings.TrimSpace(nh.Brand)
	nh.Description = trimmedOrNil(nh.Description)
	nh.PurchaseDate = trimmedOrNil(nh.PurchaseDate)
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
