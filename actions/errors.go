package actions

import "errors"

var (
	// ErrUserExists is returned if a user tries to register an existing user
	ErrUserExists = errors.New("user already registered in manager")
)
