// Package users manages the users allowed to play games
package users

import "fmt"

// User is a user of the system
type User struct {
	Bank int64

	Name     string
	Username string
	Token    string
}

// Load returns a user based on the username and token
func Load(username string, token string) (User, error) {

	// TODO: read from external database
	return loadFromStatic(username, token)

}

// load loads from the external db
func loadFromStatic(username, token string) (User, error) {

	var u User
	var ok bool

	if u, ok = userdb[username]; !ok {
		return User{}, fmt.Errorf("invalid login for [%v]", username)
	}

	if u.Token != token {
		return User{}, fmt.Errorf("invalid login for [%v] (wrong password)", username)
	}
	return u, nil
}

// Check returns true if the username and token are valid
func Check(username, token string) bool {
	if _, ok := userdb[username]; !ok {
		return false
	}
	return true
}
