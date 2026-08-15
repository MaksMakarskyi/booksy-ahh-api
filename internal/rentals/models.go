package rentals

type Rental struct {
	ID         int     `json:"id"`
	UserID     int     `json:"user_id"`
	HardwareID int     `json:"hardware_id"`
	RentedAt   string  `json:"rented_at"`
	ReturnedAt *string `json:"returned_at"`
}

type RentalWithHardware struct {
	ID          int     `json:"id"`
	UserID      int     `json:"user_id"`
	HardwareID  int     `json:"hardware_id"`
	Name        string  `json:"name"`
	Brand       string  `json:"brand"`
	Description *string `json:"description"`
	RentedAt    string  `json:"rented_at"`
	ReturnedAt  *string `json:"returned_at"`
}
