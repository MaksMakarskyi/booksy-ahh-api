package tests

import (
	"fmt"
	"testing"
)

func TestOnlyTheAdminExistsOnAFreshInstall(t *testing.T) {
	a := newAPI(t)

	status, body := a.call(a.admin, "GET", "/profiles", "")
	if status != 200 {
		t.Fatalf("status = %d, want 200 (%s)", status, body)
	}
	if got := count(t, body, "data"); got != 1 {
		t.Fatalf("got %d profiles, want only the bootstrapped admin (%s)", got, body)
	}
	if got := field(t, body, "data.0.email"); got != adminEmail {
		t.Errorf("email = %v, want %s", got, adminEmail)
	}
}

func TestProfilesAreAdminOnly(t *testing.T) {
	a := newAPI(t)
	employee, id := a.employee("notadmin@booksy.com")

	tests := []struct {
		name, method, path, body string
	}{
		{"list", "GET", "/profiles", ""},
		{"create", "POST", "/profiles",
			`{"email":"x@booksy.com","password":"Str0ngPass!","full_name":"X Y","role":"employee"}`},
		{"delete", "DELETE", fmt.Sprintf("/profiles/%d", id), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call(employee, tt.method, tt.path, tt.body)
			if status != 403 {
				t.Errorf("status = %d, want 403 (%s)", status, body)
			}
		})
	}
}

func TestCreateProfile(t *testing.T) {
	a := newAPI(t)

	status, body := a.call(a.admin, "POST", "/profiles", `{
		"email":     "new.hire@booksy.com",
		"password":  "Str0ngPass!",
		"full_name": "New Hire",
		"role":      "employee"
	}`)
	if status != 201 {
		t.Fatalf("status = %d, want 201 (%s)", status, body)
	}
	if got := field(t, body, "data.email"); got != "new.hire@booksy.com" {
		t.Errorf("email = %v", got)
	}

	for _, key := range []string{"password", "password_hash", "PasswordHash"} {
		if field(t, body, "data."+key) != nil {
			t.Errorf("response leaked %q: %s", key, body)
		}
	}

	a.login("new.hire@booksy.com", userPassword)
}

func TestOnlyBooksyEmailsAreAccepted(t *testing.T) {
	a := newAPI(t)

	tests := []struct {
		email string
		want  int
	}{
		{"someone@gmail.com", 400},
		{"someone@booksy.co", 400},
		{"someone@notbooksy.com", 400},
		{"someone@booksy.com.evil.com", 400},
		{"Loud.Person@BOOKSY.COM", 201},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			status, body := a.call(a.admin, "POST", "/profiles", fmt.Sprintf(
				`{"email":%q,"password":%q,"full_name":"Some One","role":"employee"}`,
				tt.email, userPassword))
			if status != tt.want {
				t.Fatalf("status = %d, want %d (%s)", status, tt.want, body)
			}

			if tt.want == 400 {
				if got := field(t, body, "error.fields.0.rule"); got != "endswith" {
					t.Errorf("rejected by rule %v, want endswith (%s)", got, body)
				}
			} else if got := field(t, body, "data.email"); got != "loud.person@booksy.com" {
				t.Errorf("email = %v, want it stored lowercased", got)
			}
		})
	}
}

func TestCreateProfileValidation(t *testing.T) {
	a := newAPI(t)

	tests := []struct {
		name, body, wantField string
	}{
		{"not an email",
			`{"email":"nope","password":"Str0ngPass!","full_name":"Valid Person","role":"employee"}`, "email"},
		{"password too short",
			`{"email":"v@booksy.com","password":"Ab1!","full_name":"Valid Person","role":"employee"}`, "password"},
		{"password without an uppercase letter",
			`{"email":"v@booksy.com","password":"str0ngpass!","full_name":"Valid Person","role":"employee"}`, "password"},
		{"password without a digit",
			`{"email":"v@booksy.com","password":"StrongPass!","full_name":"Valid Person","role":"employee"}`, "password"},
		{"password without a symbol",
			`{"email":"v@booksy.com","password":"Str0ngPass1","full_name":"Valid Person","role":"employee"}`, "password"},
		{"name too short",
			`{"email":"v@booksy.com","password":"Str0ngPass!","full_name":"A","role":"employee"}`, "full_name"},
		{"unknown role",
			`{"email":"v@booksy.com","password":"Str0ngPass!","full_name":"Valid Person","role":"superuser"}`, "role"},
		{"missing role",
			`{"email":"v@booksy.com","password":"Str0ngPass!","full_name":"Valid Person"}`, "role"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call(a.admin, "POST", "/profiles", tt.body)
			if status != 400 {
				t.Fatalf("status = %d, want 400 (%s)", status, body)
			}
			if got := field(t, body, "error.fields.0.field"); got != tt.wantField {
				t.Errorf("rejected field = %v, want %s (%s)", got, tt.wantField, body)
			}
		})
	}
}

func TestDuplicateEmailIsAConflict(t *testing.T) {
	a := newAPI(t)
	a.employee("dup@booksy.com")

	status, body := a.call(a.admin, "POST", "/profiles", fmt.Sprintf(
		`{"email":"dup@booksy.com","password":%q,"full_name":"Dup Person","role":"employee"}`,
		userPassword))
	if status != 409 {
		t.Errorf("status = %d, want 409 (%s)", status, body)
	}
}

func TestDeleteProfile(t *testing.T) {
	a := newAPI(t)
	_, id := a.employee("gone@booksy.com")
	path := fmt.Sprintf("/profiles/%d", id)

	if status, body := a.call(a.admin, "DELETE", path, ""); status != 200 {
		t.Fatalf("delete: status = %d, want 200 (%s)", status, body)
	}

	tests := []struct {
		name, path string
		want       int
	}{
		{"delete twice", path, 404},
		{"no such profile", "/profiles/999999", 404},
		{"id is not a number", "/profiles/abc", 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call(a.admin, "DELETE", tt.path, "")
			if status != tt.want {
				t.Errorf("status = %d, want %d (%s)", status, tt.want, body)
			}
		})
	}
}

func TestCannotDeleteAProfileWithRentalHistory(t *testing.T) {
	a := newAPI(t)
	employee, employeeID := a.employee("hashistory@booksy.com")

	rental := a.rent(employee, a.device("Historic"))
	if status, body := a.call(employee, "PATCH", fmt.Sprintf("/rentals/%d/return", rental), ""); status != 200 {
		t.Fatalf("return: status = %d, want 200 (%s)", status, body)
	}

	status, body := a.call(a.admin, "DELETE", fmt.Sprintf("/profiles/%d", employeeID), "")
	if status != 409 {
		t.Errorf("status = %d, want 409 (%s)", status, body)
	}
}
