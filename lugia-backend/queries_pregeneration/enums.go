package queries_pregeneration

type UserRole string

const (
	AdminRole  UserRole = "admin"
	EditorRole UserRole = "editor"
)

func (ur UserRole) String() string {
	return string(ur)
}
