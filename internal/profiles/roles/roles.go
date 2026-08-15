package roles

type Role string

const (
	Employee Role = "employee"
	Admin    Role = "admin"
)

func (r Role) IsValid() bool {
	switch r {
	case Employee, Admin:
		return true
	default:
		return false
	}
}
