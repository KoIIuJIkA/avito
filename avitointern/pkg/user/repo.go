package user

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrNoUser  = errors.New("No user found")
	ErrBadPass = errors.New("Invald password")
)

var _ UserRepo = &UserMemoryRepository{}

type UserMemoryRepository struct {
	data map[string]*User
}

func NewMemoryRepo() *UserMemoryRepository {
	return &UserMemoryRepository{
		data: map[string]*User{
			"george": &User{
				ID:             uuid.New().String(),
				Username:       "george",
				FirstName:      "George",
				LastName:       "Original",
				Password:       "qwer",
				OrganizationID: "123e4567-e89b-12d3-a456-426614174000",
			},
		},
	}
}

func (repo *UserMemoryRepository) Authorize(username, pass string) (*User, error) {
	u, ok := repo.data[username]
	if !ok {
		return nil, ErrNoUser
	}

	// piu pau authorization
	if u.Password != pass {
		return nil, ErrBadPass
	}

	return u, nil
}

func (repo *UserMemoryRepository) GetUserByID(userID string) (*User, error) {
	for _, user := range repo.data {
		if user.ID == userID {
			return user, nil
		}
	}
	return nil, ErrNoUser
}
