package pokerclient

import (
	"context"

	"golang.org/x/oauth2"
)

const (
	authURL  = "https://login.wetsnow.com/auth/realms/wetsnow/protocol/openid-connect/auth"
	tokenURL = "https://login.wetsnow.com/auth/realms/wetsnow/protocol/openid-connect/token"
)

func oauthClientConfig() *oauth2.Config {
	conf := &oauth2.Config{
		ClientID: "pepper-poker-grpc.wetsnow.com",
		// ClientSecret: "YOUR_CLIENT_SECRET",
		Scopes: []string{"openid"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
	return conf
}

func getAuthToken(ctx context.Context, username, password string) (*oauth2.Token, error) {

	conf := oauthClientConfig()
	return conf.PasswordCredentialsToken(ctx, username, password)
}
