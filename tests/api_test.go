package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/MaksMakarskyi/booksy-go-api/internal/profiles"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server/config"
	"github.com/MaksMakarskyi/booksy-go-api/internal/server/dependencies"
	"github.com/MaksMakarskyi/booksy-go-api/internal/utils/migrate"
)

const (
	adminEmail    = "admin@booksy.com"
	adminPassword = "Adm1nPass!"
	userPassword  = "Str0ngPass!"
)

type api struct {
	t       *testing.T
	handler http.Handler
	admin   string
}

func newAPI(t *testing.T, tweak ...func(*config.Config)) *api {
	t.Helper()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	cfg := &config.Config{
		Env:          config.Production,
		Port:         "0",
		DatabaseUrl:  "file:" + dbPath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)",
		JWTSecret:    "test-jwt-secret-with-at-least-32-bytes",
		JWTTTL:       time.Hour,
		CORSOrigins:  []string{"*"},
		RateLimitRPS: 10_000,
		GooseTable:   "goose_migrations",
		Admins: config.AdminAccounts{
			{Email: adminEmail, Password: adminPassword, FullName: "Test Admin"},
		},
	}
	for _, apply := range tweak {
		apply(cfg)
	}

	deps, err := dependencies.NewRegistry(ctx, cfg)
	if err != nil {
		t.Fatalf("build dependencies: %v", err)
	}
	t.Cleanup(func() {
		_ = deps.Close()
	})

	if err := migrate.Up(deps.DB, cfg.GooseTable); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := profiles.EnsureAdmins(ctx, deps); err != nil {
		t.Fatalf("ensure admins: %v", err)
	}

	handler, err := server.NewServer(deps)
	if err != nil {
		t.Fatalf("build server: %v", err)
	}

	a := &api{t: t, handler: handler}
	a.admin = a.login(adminEmail, adminPassword)

	return a
}

func (a *api) call(token, method, path, body string) (status int, response string) {
	a.t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)

	return rec.Code, rec.Body.String()
}

func (a *api) login(email, password string) string {
	a.t.Helper()

	status, body := a.call("", "POST", "/auth/token",
		fmt.Sprintf(`{"email":%q,"password":%q}`, email, password))
	if status != http.StatusOK {
		a.t.Fatalf("login %s: %d %s", email, status, body)
	}

	token, _ := field(a.t, body, "data.access_token").(string)
	if token == "" {
		a.t.Fatalf("login %s: no access_token in %s", email, body)
	}

	return token
}

func (a *api) employee(email string) (token string, id int) {
	a.t.Helper()

	status, body := a.call(a.admin, "POST", "/profiles", fmt.Sprintf(
		`{"email":%q,"password":%q,"full_name":"Test Employee","role":"employee"}`,
		email, userPassword))
	if status != http.StatusCreated {
		a.t.Fatalf("create employee %s: %d %s", email, status, body)
	}

	return a.login(email, userPassword), field(a.t, body, "data.id").(int)
}

func (a *api) device(name string) int {
	a.t.Helper()

	status, body := a.call(a.admin, "POST", "/hardware",
		fmt.Sprintf(`{"name":%q,"brand":"TestCo"}`, name))
	if status != http.StatusCreated {
		a.t.Fatalf("create hardware %s: %d %s", name, status, body)
	}

	return field(a.t, body, "data.id").(int)
}

func (a *api) rent(token string, deviceID int) int {
	a.t.Helper()

	status, body := a.call(token, "POST", "/rentals",
		fmt.Sprintf(`{"hardware_id":%d}`, deviceID))
	if status != http.StatusCreated {
		a.t.Fatalf("rent device %d: %d %s", deviceID, status, body)
	}

	return field(a.t, body, "data.id").(int)
}

func field(t *testing.T, body, path string) any {
	t.Helper()

	var value any
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		t.Fatalf("body is not JSON: %s", body)
	}

	for key := range strings.SplitSeq(path, ".") {
		switch node := value.(type) {
		case map[string]any:
			value = node[key]
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i >= len(node) {
				return nil
			}
			value = node[i]
		default:
			return nil
		}
	}

	if number, ok := value.(float64); ok && number == math.Trunc(number) {
		return int(number)
	}

	return value
}

func count(t *testing.T, body, path string) int {
	t.Helper()

	items, ok := field(t, body, path).([]any)
	if !ok {
		t.Fatalf("%s is not an array in %s", path, body)
	}

	return len(items)
}
