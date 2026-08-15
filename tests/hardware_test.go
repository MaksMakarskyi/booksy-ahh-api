package tests

import (
	"fmt"
	"testing"
)

func TestSeededInventory(t *testing.T) {
	a := newAPI(t)

	status, body := a.call(a.admin, "GET", "/hardware", "")
	if status != 200 {
		t.Fatalf("status = %d, want 200 (%s)", status, body)
	}

	if got := count(t, body, "data"); got != 11 {
		t.Fatalf("got %d devices, want the 11 seeded ones", got)
	}

	for i := range count(t, body, "data") {
		if got := field(t, body, fmt.Sprintf("data.%d.status", i)); got == "in_use" {
			t.Errorf("device %v is seeded as in_use with no rental to close it",
				field(t, body, fmt.Sprintf("data.%d.id", i)))
		}
	}
}

func TestEmployeeCanReadButNotWriteHardware(t *testing.T) {
	a := newAPI(t)
	employee, _ := a.employee("reader@booksy.com")

	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"list", "GET", "/hardware", "", 200},
		{"create", "POST", "/hardware", `{"name":"Nope","brand":"X"}`, 403},
		{"update", "PATCH", "/hardware/1", `{"name":"Nope"}`, 403},
		{"delete", "DELETE", "/hardware/1", "", 403},
		{"send to repair", "PATCH", "/hardware/1/repair", "", 403},
		{"back to stock", "PATCH", "/hardware/3/available", "", 403},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call(employee, tt.method, tt.path, tt.body)
			if status != tt.want {
				t.Errorf("status = %d, want %d (%s)", status, tt.want, body)
			}
		})
	}
}

func TestCreateHardware(t *testing.T) {
	a := newAPI(t)

	status, body := a.call(a.admin, "POST", "/hardware", `{
		"name":          "Framework 13",
		"brand":         "Framework",
		"description":   "Spare loaner",
		"purchase_date": "2024-03-15"
	}`)
	if status != 201 {
		t.Fatalf("status = %d, want 201 (%s)", status, body)
	}

	if got := field(t, body, "data.name"); got != "Framework 13" {
		t.Errorf("name = %v, want Framework 13", got)
	}
	if got := field(t, body, "data.status"); got != "available" {
		t.Errorf("status = %v, want available", got)
	}
}

func TestCreateHardwareValidation(t *testing.T) {
	a := newAPI(t)

	tests := []struct {
		name, body, wantField string
	}{
		{"missing name", `{"brand":"X"}`, "name"},
		{"blank name", `{"name":"   ","brand":"X"}`, "name"},
		{"missing brand", `{"name":"X"}`, "brand"},
		{"purchased in the future", `{"name":"X","brand":"Y","purchase_date":"2099-01-01"}`, "purchase_date"},
		{"wrong date format", `{"name":"X","brand":"Y","purchase_date":"15-03-2024"}`, "purchase_date"},
		{"timestamp is not a date", `{"name":"X","brand":"Y","purchase_date":"2024-03-15T00:00:00Z"}`, "purchase_date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call(a.admin, "POST", "/hardware", tt.body)
			if status != 400 {
				t.Fatalf("status = %d, want 400 (%s)", status, body)
			}
			if got := field(t, body, "error.fields.0.field"); got != tt.wantField {
				t.Errorf("rejected field = %v, want %s (%s)", got, tt.wantField, body)
			}
		})
	}
}

func TestUpdateHardware(t *testing.T) {
	a := newAPI(t)

	tests := []struct {
		name, body string
		want       int
	}{
		{"rename", `{"name":"After"}`, 200},
		{"several fields at once", `{"name":"After","brand":"Other","description":"Note"}`, 200},
		{"nothing to change", `{}`, 400},
		{"blank name", `{"name":"   "}`, 400},
		{"status is not a client field", `{"status":"in_use"}`, 400},
		{"misspelled field", `{"nmae":"typo"}`, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := a.device("Before")

			status, body := a.call(a.admin, "PATCH", fmt.Sprintf("/hardware/%d", id), tt.body)
			if status != tt.want {
				t.Fatalf("status = %d, want %d (%s)", status, tt.want, body)
			}
			if status == 200 && field(t, body, "data.name") != "After" {
				t.Errorf("name = %v, want After", field(t, body, "data.name"))
			}
		})
	}
}

func TestDeleteHardware(t *testing.T) {
	a := newAPI(t)
	id := a.device("Doomed")
	path := fmt.Sprintf("/hardware/%d", id)

	if status, body := a.call(a.admin, "DELETE", path, ""); status != 200 {
		t.Fatalf("delete: status = %d, want 200 (%s)", status, body)
	}
	if status, body := a.call(a.admin, "DELETE", path, ""); status != 404 {
		t.Errorf("delete twice: status = %d, want 404 (%s)", status, body)
	}
}

func TestHardwarePathParameters(t *testing.T) {
	a := newAPI(t)

	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"unknown id on update", "PATCH", "/hardware/999999", `{"name":"Ghost"}`, 404},
		{"unknown id on delete", "DELETE", "/hardware/999999", "", 404},
		{"unknown id on repair", "PATCH", "/hardware/999999/repair", "", 404},
		{"id is not a number", "DELETE", "/hardware/abc", "", 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call(a.admin, tt.method, tt.path, tt.body)
			if status != tt.want {
				t.Errorf("status = %d, want %d (%s)", status, tt.want, body)
			}
		})
	}
}

func TestRepairTransitions(t *testing.T) {
	a := newAPI(t)
	id := a.device("Repairable")

	steps := []struct {
		name, path string
		want       int
		wantStatus string
	}{
		{"available -> repair", "repair", 200, "repair"},
		{"repair -> repair", "repair", 409, ""},
		{"repair -> available", "available", 200, "available"},
		{"available -> available", "available", 409, ""},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			status, body := a.call(a.admin, "PATCH",
				fmt.Sprintf("/hardware/%d/%s", id, step.path), "")
			if status != step.want {
				t.Fatalf("status = %d, want %d (%s)", status, step.want, body)
			}
			if step.wantStatus != "" && field(t, body, "data.status") != step.wantStatus {
				t.Errorf("status = %v, want %s", field(t, body, "data.status"), step.wantStatus)
			}
		})
	}
}

func TestRentedHardwareCannotGoToRepair(t *testing.T) {
	a := newAPI(t)
	employee, _ := a.employee("renter@booksy.com")
	id := a.device("In Use Soon")

	a.rent(employee, id)

	status, body := a.call(a.admin, "PATCH", fmt.Sprintf("/hardware/%d/repair", id), "")
	if status != 409 {
		t.Errorf("status = %d, want 409 (%s)", status, body)
	}
}
