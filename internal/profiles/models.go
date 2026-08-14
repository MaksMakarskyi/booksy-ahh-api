package profiles

type Profile struct {
	ID        int         `json:"id"`
	Email     string      `json:"email"`
	FullName  string      `json:"full_name"`
	Role      ProfileRole `json:"role"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
}

type ProfileRole string

const (
	Employee ProfileRole = "employee"
	Admin    ProfileRole = "admin"
)

// IsValid reports whether the role is one this application recognises.
//
// Request payloads are covered by the `oneof` struct tag; this exists for
// values arriving from outside the validator — currently the role claim in a
// JWT, which is a plain string by the time it crosses the jwt package's
// domain-free boundary.
func (pr ProfileRole) IsValid() bool {
	switch pr {
	case Employee, Admin:
		return true
	default:
		return false
	}
}

type NewProfile struct {
	Email        string
	FullName     string
	Role         ProfileRole
	PasswordHash string
}
