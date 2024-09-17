package user

type User struct {
	ID             string
	Username       string
	FirstName      string
	LastName       string
	Password       string
	OrganizationID string
}

type UserRepo interface {
	Authorize(username, pass string) (*User, error)
	GetUserByID(userID string) (*User, error)
}
