package tests

import (
	"fmt"
	"testing"
)

func TestRentalsStartEmpty(t *testing.T) {
	a := newAPI(t)

	status, body := a.call(a.admin, "GET", "/rentals", "")
	if status != 200 {
		t.Fatalf("status = %d, want 200 (%s)", status, body)
	}
	if got := count(t, body, "data"); got != 0 {
		t.Errorf("got %d seeded rentals, want none (%s)", got, body)
	}
}

func TestRentAndReturn(t *testing.T) {
	a := newAPI(t)
	employee, employeeID := a.employee("rt@booksy.com")
	deviceID := a.device("Round Trip")

	status, body := a.call(employee, "POST", "/rentals", fmt.Sprintf(`{"hardware_id":%d}`, deviceID))
	if status != 201 {
		t.Fatalf("rent: status = %d, want 201 (%s)", status, body)
	}
	if got := field(t, body, "data.user_id"); got != employeeID {
		t.Errorf("user_id = %v, want %d", got, employeeID)
	}
	if got := field(t, body, "data.returned_at"); got != nil {
		t.Errorf("a new rental must be open, got returned_at = %v", got)
	}
	rentalID := field(t, body, "data.id").(int)

	if got := deviceStatus(t, a, employee, deviceID); got != "in_use" {
		t.Errorf("device status = %v, want in_use", got)
	}

	status, body = a.call(employee, "PATCH", fmt.Sprintf("/rentals/%d/return", rentalID), "")
	if status != 200 {
		t.Fatalf("return: status = %d, want 200 (%s)", status, body)
	}
	if field(t, body, "data.returned_at") == nil {
		t.Error("returned_at was not set")
	}

	if got := deviceStatus(t, a, employee, deviceID); got != "available" {
		t.Errorf("device status after return = %v, want available", got)
	}
	a.rent(employee, deviceID)
}

func TestRentRejectsUnrentableDevices(t *testing.T) {
	a := newAPI(t)
	employee, _ := a.employee("rentval@booksy.com")

	taken := a.device("Contested")
	other, _ := a.employee("holder@booksy.com")
	a.rent(other, taken)

	broken := a.device("Broken")
	if status, body := a.call(a.admin, "PATCH", fmt.Sprintf("/hardware/%d/repair", broken), ""); status != 200 {
		t.Fatalf("send to repair: status = %d (%s)", status, body)
	}

	tests := []struct {
		name, body string
		want       int
	}{
		{"already rented", fmt.Sprintf(`{"hardware_id":%d}`, taken), 409},
		{"in repair", fmt.Sprintf(`{"hardware_id":%d}`, broken), 409},
		{"no such device", `{"hardware_id":999999}`, 404},
		{"id must be positive", `{"hardware_id":0}`, 400},
		{"missing id", `{}`, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call(employee, "POST", "/rentals", tt.body)
			if status != tt.want {
				t.Errorf("status = %d, want %d (%s)", status, tt.want, body)
			}
		})
	}
}

func TestRentalsAreScopedToTheirOwner(t *testing.T) {
	a := newAPI(t)
	owner, _ := a.employee("owner@booksy.com")
	other, _ := a.employee("other@booksy.com")

	ownerRental := a.rent(owner, a.device("Owned"))
	adminRental := a.rent(a.admin, a.device("Admins"))

	t.Run("listing shows only your own", func(t *testing.T) {
		tests := []struct {
			name, token string
			wantRental  int
		}{
			{"owner", owner, ownerRental},
			{"admin", a.admin, adminRental},
			{"uninvolved employee", other, 0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, body := a.call(tt.token, "GET", "/rentals", "")

				if tt.wantRental == 0 {
					if got := count(t, body, "data"); got != 0 {
						t.Fatalf("sees %d rentals, want none (%s)", got, body)
					}
					return
				}
				if got := count(t, body, "data"); got != 1 {
					t.Fatalf("sees %d rentals, want exactly 1 (%s)", got, body)
				}
				if got := field(t, body, "data.0.id"); got != tt.wantRental {
					t.Errorf("sees rental %v, want %d", got, tt.wantRental)
				}
			})
		}
	})

	t.Run("returning someone else's is forbidden", func(t *testing.T) {
		tests := []struct {
			name, token string
			rental      int
		}{
			{"another employee", other, ownerRental},
			{"the admin", a.admin, ownerRental},
			{"an employee returning the admin's", owner, adminRental},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				status, body := a.call(tt.token, "PATCH",
					fmt.Sprintf("/rentals/%d/return", tt.rental), "")
				if status != 403 {
					t.Errorf("status = %d, want 403 (%s)", status, body)
				}
			})
		}
	})
}

func TestReturnRejectsBadRequests(t *testing.T) {
	a := newAPI(t)
	employee, _ := a.employee("twice@booksy.com")

	rental := a.rent(employee, a.device("Returned Twice"))
	if status, body := a.call(employee, "PATCH", fmt.Sprintf("/rentals/%d/return", rental), ""); status != 200 {
		t.Fatalf("first return: status = %d, want 200 (%s)", status, body)
	}

	tests := []struct {
		name, path string
		want       int
	}{
		{"already returned", fmt.Sprintf("/rentals/%d/return", rental), 409},
		{"no such rental", "/rentals/999999/return", 404},
		{"id is not a number", "/rentals/abc/return", 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call(employee, "PATCH", tt.path, "")
			if status != tt.want {
				t.Errorf("status = %d, want %d (%s)", status, tt.want, body)
			}
		})
	}
}

func deviceStatus(t *testing.T, a *api, token string, deviceID int) any {
	t.Helper()

	_, body := a.call(token, "GET", "/hardware", "")
	for i := range count(t, body, "data") {
		if field(t, body, fmt.Sprintf("data.%d.id", i)) == deviceID {
			return field(t, body, fmt.Sprintf("data.%d.status", i))
		}
	}

	t.Fatalf("device %d is not in the listing: %s", deviceID, body)

	return nil
}
