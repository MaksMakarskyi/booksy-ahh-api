package tests

import (
	"testing"

	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
)

func TestLogin(t *testing.T) {
	a := newAPI(t)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"bootstrapped admin", `{"email":"admin@booksy.com","password":"Adm1nPass!"}`, 200},
		{"wrong password", `{"email":"admin@booksy.com","password":"Wr0ngPass!"}`, 401},
		{"unknown email", `{"email":"nobody@booksy.com","password":"Adm1nPass!"}`, 401},
		{"empty payload", `{}`, 400},
		{"broken json", `{oops`, 400},
		{"unknown field", `{"email":"admin@booksy.com","password":"Adm1nPass!","extra":true}`, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := a.call("", "POST", "/auth/token", tt.body)
			if status != tt.want {
				t.Fatalf("status = %d, want %d (%s)", status, tt.want, body)
			}
		})
	}
}

func TestLoginReturnsUsableToken(t *testing.T) {
	a := newAPI(t)

	status, body := a.call("", "POST", "/auth/token",
		`{"email":"admin@booksy.com","password":"Adm1nPass!"}`)
	if status != 200 {
		t.Fatalf("status = %d, want 200 (%s)", status, body)
	}

	if got := field(t, body, "data.token_type"); got != "Bearer" {
		t.Errorf("token_type = %v, want Bearer", got)
	}
	if field(t, body, "data.expires_at") == nil {
		t.Error("no expires_at in the response")
	}

	token, _ := field(t, body, "data.access_token").(string)
	if status, body := a.call(token, "GET", "/hardware", ""); status != 200 {
		t.Errorf("the issued token was rejected: %d %s", status, body)
	}
}

func TestLoginFailuresAreIndistinguishable(t *testing.T) {
	a := newAPI(t)

	_, wrongPassword := a.call("", "POST", "/auth/token",
		`{"email":"admin@booksy.com","password":"Wr0ngPass!"}`)
	_, unknownEmail := a.call("", "POST", "/auth/token",
		`{"email":"nobody@booksy.com","password":"Adm1nPass!"}`)

	got, want := field(t, wrongPassword, "error.message"), field(t, unknownEmail, "error.message")
	if got != want {
		t.Errorf("wrong password says %q, unknown email says %q", got, want)
	}
}

func TestProtectedRoutesRejectBadTokens(t *testing.T) {
	a := newAPI(t)

	tests := []struct {
		name  string
		token string
	}{
		{"no token", ""},
		{"not a jwt", "garbage"},
		{"foreign signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
			"eyJlbWFpbCI6ImFAYi5jbyIsIm5hbWUiOiJBIiwicm9sZSI6ImFkbWluIiwic3ViIjoiMSIsImV4cCI6OTk5OTk5OTk5OX0." +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if status, body := a.call(tt.token, "GET", "/hardware", ""); status != 401 {
				t.Errorf("status = %d, want 401 (%s)", status, body)
			}
		})
	}
}

func TestHealthzIsPublic(t *testing.T) {
	a := newAPI(t)

	if status, body := a.call("", "GET", "/healthz", ""); status != 200 {
		t.Errorf("status = %d, want 200 (%s)", status, body)
	}
}

func TestRateLimitRejectsABurst(t *testing.T) {
	a := newAPI(t, func(c *config.Config) {
		c.RateLimitRPS = 5
	})

	for i := range 20 {
		status, body := a.call("", "GET", "/healthz", "")
		if status == 429 {
			return
		}
		if status != 200 {
			t.Fatalf("request %d: status = %d, want 200 or 429 (%s)", i, status, body)
		}
	}

	t.Error("20 requests in a burst were all accepted, the rate limit is not applied")
}
