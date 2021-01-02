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
}

// Load returns a user based on the username and token
func Load(username string) (User, error) {

	// TODO: read from external database
	return loadFromStatic(username)

}

// load loads from the external db
func loadFromStatic(username string) (User, error) {

	if !Check(username) {
		return User{}, fmt.Errorf("invalid user [%v]", username)
	}

	return userdb[username], nil
}

// Check returns true if the username is a valid user
func Check(username string) bool {
	if _, ok := userdb[username]; !ok {
		authchecks.WithLabelValues("failure").Inc()
		return false
	}
	authchecks.WithLabelValues("success").Inc()
	return true
}
