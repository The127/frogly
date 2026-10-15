package repositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// User is a person who has used frogly. The name and email are a copy of
// what the person's login says, and Refresh keeps them up to date.
type User struct {
	Base

	name  string
	email string
}

// NewUser creates a user who was never saved. A user needs a name.
func NewUser(name string, email string) (*User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: a user needs a name", ErrInvalid)
	}

	return &User{
		Base:  newBase(),
		name:  name,
		email: email,
	}, nil
}

// NewUserFromDB rebuilds a user as loaded.
func NewUserFromDB(base Base, name string, email string) *User {
	return &User{Base: base, name: name, email: email}
}

// Name is how frogly greets the user.
func (user *User) Name() string {
	return user.name
}

// Email is where frogly could reach the user, empty when unknown.
func (user *User) Email() string {
	return user.email
}

// Refresh takes the name and email from the user's latest login and says
// whether anything changed. An empty name keeps the old one, since a user
// always needs one.
func (user *User) Refresh(name string, email string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		name = user.name
	}

	if name == user.name && email == user.email {
		return false
	}

	user.name = name
	user.email = email

	return true
}

// UserRepository loads and saves users.
type UserRepository interface {
	First(ctx context.Context, id uuid.UUID) (*User, error)
	Insert(user *User)
	Update(user *User)
}
