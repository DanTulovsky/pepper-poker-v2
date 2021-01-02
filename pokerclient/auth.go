package pokerclient

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/DanTulovsky/pepper-poker-v2/auth"
)

// getAuthToken gets an auth token based on the username and password of the client
func getAuthToken(ctx context.Context, username, password string) (*oauth2.Token, error) {
	conf := auth.OAuthClientConfig()
	return conf.PasswordCredentialsToken(ctx, username, password)
}
