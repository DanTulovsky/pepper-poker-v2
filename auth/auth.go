package auth

import "golang.org/x/oauth2"

const (
	// OIDCProviderURL is the root url of the auth provider
	OIDCProviderURL = "https://login.wetsnow.com"

	// Realm is the auth realm
	Realm = "wetsnow"

	// Audience is the required audience
	Audience = "pepper-poker-grpc"

	// ClientID is the oauth client id
	ClientID = "pepper-poker-grpc.wetsnow.com"

	// Issuer is the cert issuer
	Issuer = "https://login.wetsnow.com/auth/realms/wetsnow"

	// AuthURL is the auth URL
	AuthURL = "https://login.wetsnow.com/auth/realms/wetsnow/protocol/openid-connect/auth"

	// TokenURL is the token URL
	TokenURL = "https://login.wetsnow.com/auth/realms/wetsnow/protocol/openid-connect/token"
)

// OAuthClientConfig returns the oauth client config for retrieving tokens
func OAuthClientConfig() *oauth2.Config {
	conf := &oauth2.Config{
		ClientID: ClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  AuthURL,
			TokenURL: TokenURL,
		},
	}
	return conf
}
