package rentals

type CreateRentalReq struct {
	HardwareID int `json:"hardware_id" validate:"required,gt=0"`
}
