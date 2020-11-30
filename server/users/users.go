// Package users manages the users allowed to play games
package users

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	authchecks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pepperpoker_authchecks_total",
		Help: "keeps track of failed and successful auth checks",
	}, []string{"result"})
)

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

	if !Check(username, token) {
		return User{}, fmt.Errorf("invalid login for [%v]", username)
	}

	return userdb[username], nil
}

// Check returns true if the username and token are valid
func Check(username, token string) bool {
	if _, ok := userdb[username]; !ok {
		authchecks.WithLabelValues("failure").Inc()
		return false
	}
	authchecks.WithLabelValues("success").Inc()
	return true
}
