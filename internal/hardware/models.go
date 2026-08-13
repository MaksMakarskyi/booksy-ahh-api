package hardware

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
