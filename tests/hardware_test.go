package tests

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
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

func TestClearableFieldsDistinguishEmptyFromAbsent(t *testing.T) {
	a := newAPI(t)

	fill := `{"description":"Some note","purchase_date":"2024-03-15"}`

	tests := []struct {
		name, body, wantDescription, wantPurchaseDate string
	}{
		{"clear the description", `{"description":""}`, "", "2024-03-15"},
		{"clear the purchase date", `{"purchase_date":""}`, "Some note", ""},
		{"clear both at once", `{"description":"","purchase_date":""}`, "", ""},
		{"whitespace clears too", `{"description":"   "}`, "", "2024-03-15"},
		{"an absent key changes nothing", `{"name":"Renamed"}`, "Some note", "2024-03-15"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := a.device("Clearable")
			path := fmt.Sprintf("/hardware/%d", id)

			if status, body := a.call(a.admin, "PATCH", path, fill); status != 200 {
				t.Fatalf("fill: status = %d, want 200 (%s)", status, body)
			}

			status, body := a.call(a.admin, "PATCH", path, tt.body)
			if status != 200 {
				t.Fatalf("status = %d, want 200 (%s)", status, body)
			}
			if got := field(t, body, "data.description"); got != tt.wantDescription {
				t.Errorf("description = %q, want %q", got, tt.wantDescription)
			}
			if got := field(t, body, "data.purchase_date"); got != tt.wantPurchaseDate {
				t.Errorf("purchase_date = %q, want %q", got, tt.wantPurchaseDate)
			}

			// The cleared value must survive a re-read, not just the RETURNING row.
			_, list := a.call(a.admin, "GET", "/hardware", "")
			for i := range count(t, list, "data") {
				if field(t, list, fmt.Sprintf("data.%d.id", i)) != id {
					continue
				}
				if got := field(t, list, fmt.Sprintf("data.%d.description", i)); got != tt.wantDescription {
					t.Errorf("description after re-read = %q, want %q", got, tt.wantDescription)
				}
			}
		})
	}
}

// name and brand back NOT NULL columns and carry meaning, so clearing them is
// a mistake rather than an erasure.
func TestNameAndBrandCannotBeCleared(t *testing.T) {
	a := newAPI(t)

	tests := []struct {
		name, body, wantField string
	}{
		{"empty name", `{"name":""}`, "name"},
		{"blank name", `{"name":"   "}`, "name"},
		{"empty brand", `{"brand":""}`, "brand"},
		{"empty name alongside a real change", `{"name":"","description":"ok"}`, "name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := a.device("Keeps Its Name")

			status, body := a.call(a.admin, "PATCH", fmt.Sprintf("/hardware/%d", id), tt.body)
			if status != 400 {
				t.Fatalf("status = %d, want 400 (%s)", status, body)
			}
			if got := field(t, body, "error.fields.0.field"); got != tt.wantField {
				t.Errorf("rejected field = %v, want %s (%s)", got, tt.wantField, body)
			}
		})
	}
}

func TestCreateAcceptsEmptyOptionalFields(t *testing.T) {
	a := newAPI(t)

	tests := []struct {
		name, body string
	}{
		{"both keys absent", `{"name":"Bare","brand":"B"}`},
		{"both keys empty", `{"name":"Empty","brand":"B","description":"","purchase_date":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call(a.admin, "POST", "/hardware", tt.body)
			if status != 201 {
				t.Fatalf("status = %d, want 201 (%s)", status, body)
			}
			if got := field(t, body, "data.description"); got != "" {
				t.Errorf("description = %q, want empty", got)
			}
			if got := field(t, body, "data.purchase_date"); got != "" {
				t.Errorf("purchase_date = %q, want empty", got)
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

func TestSearchRequiresAQuery(t *testing.T) {
	a := newAPI(t)

	tests := []struct {
		name, path string
		want       int
	}{
		{"missing query", "/hardware/search", 400},
		{"empty query", "/hardware/search?query=", 400},
		{"whitespace only", "/hardware/search?query=%20%20", 400},
		{"valid query", "/hardware/search?query=laptop", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call(a.admin, "GET", tt.path, "")
			if status != tt.want {
				t.Errorf("status = %d, want %d (%s)", status, tt.want, body)
			}
		})
	}
}

func TestSearchReturnsAtMostTopK(t *testing.T) {
	a := newAPI(t)

	status, body := a.call(a.admin, "GET", "/hardware/search?query=apple+laptop", "")
	if status != 200 {
		t.Fatalf("status = %d, want 200 (%s)", status, body)
	}

	if got := count(t, body, "data"); got != 5 {
		t.Errorf("got %d results, want the top 5 of 11 seeded devices", got)
	}
	if field(t, body, "data.0.name") == nil {
		t.Errorf("results are not hardware objects: %s", body)
	}
}

func TestSearchDoesNotLeakVectors(t *testing.T) {
	a := newAPI(t)

	_, body := a.call(a.admin, "GET", "/hardware/search?query=phone", "")

	for _, key := range []string{"vector", "Vector", "model", "Model", "embedding"} {
		if field(t, body, "data.0."+key) != nil {
			t.Errorf("response leaked %q: %s", key, body)
		}
	}
}

func TestSearchIsAvailableToEmployees(t *testing.T) {
	a := newAPI(t)
	employee, _ := a.employee("searcher@booksy.com")

	if status, body := a.call(employee, "GET", "/hardware/search?query=mouse", ""); status != 200 {
		t.Errorf("status = %d, want 200 (%s)", status, body)
	}
	if status, body := a.call("", "GET", "/hardware/search?query=mouse", ""); status != 401 {
		t.Errorf("anonymous: status = %d, want 401 (%s)", status, body)
	}
}

func TestNewHardwareBecomesSearchable(t *testing.T) {
	a := newAPI(t, func(c *config.Config) { c.SearchTopK = 50 })

	id := a.device("Thinkpad X1 Carbon")

	_, body := a.call(a.admin, "GET", "/hardware/search?query=thinkpad", "")
	found := false
	for i := range count(t, body, "data") {
		if field(t, body, fmt.Sprintf("data.%d.id", i)) == id {
			found = true
		}
	}
	if !found {
		t.Errorf("device %d is not in the candidate set: %s", id, body)
	}
}

// Searching for a device's own text must put that device first. The fake
// embedder is character-frequency based, so an exact-text query is the closest
// possible vector to that device and nothing else can outrank it.
func TestSearchRanksTheClosestMatchFirst(t *testing.T) {
	a := newAPI(t)

	const name = "Zzyzx Quantum Widget"
	id := a.device(name)

	status, body := a.call(a.admin, "GET", "/hardware/search?query="+url.QueryEscape(name+" TestCo"), "")
	if status != 200 {
		t.Fatalf("status = %d, want 200 (%s)", status, body)
	}

	if got := field(t, body, "data.0.id"); got != id {
		t.Errorf("top match is %v (%v), want device %d",
			got, field(t, body, "data.0.name"), id)
	}
}

// The order has to depend on the query, not be fixed: searching each device's
// own text must put that device on top, so two different queries produce two
// different winners from the same candidate set.
func TestSearchOrderDependsOnTheQuery(t *testing.T) {
	a := newAPI(t)

	first := a.device("Xylophone Zephyr Quokka")
	second := a.device("Jubilant Waffle Machine")

	for _, tt := range []struct {
		query string
		want  int
	}{
		{"Xylophone Zephyr Quokka TestCo", first},
		{"Jubilant Waffle Machine TestCo", second},
	} {
		t.Run(tt.query, func(t *testing.T) {
			_, body := a.call(a.admin, "GET",
				"/hardware/search?query="+url.QueryEscape(tt.query), "")

			if got := field(t, body, "data.0.id"); got != tt.want {
				t.Errorf("top match is %v (%v), want device %d",
					got, field(t, body, "data.0.name"), tt.want)
			}
		})
	}
}
