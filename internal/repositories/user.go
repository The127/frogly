package repositories

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// User is a person who has used frogly. The name and email are a copy of
// what the person's login says, and Refresh keeps them up to date.
type User struct {
	Base

	name  string
	email string
}

// NewUser creates a user who was never saved. A user needs a name of up
// to 100 characters.
func NewUser(name string, email string) (*User, error) {
	name = strings.TrimSpace(name)
	if !ValidName(name) {
		return nil, invalid("a user needs a name of 1 to %d characters", MaxName)
	}

	return &User{
		Base:  newBase(),
		name:  name,
		email: strings.TrimSpace(email),
	}, nil
}

// ValidName says whether a trimmed name can be a user's name.
func ValidName(name string) bool {
	return name != "" && utf8.RuneCountInString(name) <= MaxName
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
// whether anything changed. A login that leaves one out, or has a name
// frogly can't use, keeps the old value. A user always needs a name, and
// an issuer may only send the email when asked for it.
func (user *User) Refresh(name string, email string) bool {
	name = strings.TrimSpace(name)
	if !ValidName(name) {
		name = user.name
	}

	email = strings.TrimSpace(email)
	if email == "" {
		email = user.email
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
	// First returns ErrNotFound when no user has the ID.
	First(ctx context.Context, id uuid.UUID) (*User, error)
	Insert(user *User)
	Update(user *User)
}
