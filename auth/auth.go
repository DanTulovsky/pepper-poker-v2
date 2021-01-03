package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/DanTulovsky/logger"
	gocloak "github.com/Nerzal/gocloak/v7"
	"github.com/davecgh/go-spew/spew"
	"github.com/dgrijalva/jwt-go/v4"
	"github.com/fatih/color"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

	clientSecretENV = "PEPPER_POKER_CLIENT_SECRET"
)

var (

	// ExpectedRoles is the list of expected roles in the oauth token
	ExpectedRoles = []string{"user"}
)

// Server is the auth modules for the server
type Server struct {
	cloakClient   gocloak.GoCloak
	ExpectedRoles []string

	l *logger.Logger
}

// NewServerClient returns a new auth server client
func NewServerClient() *Server {

	return &Server{
		cloakClient:   gocloak.NewClient(OIDCProviderURL),
		ExpectedRoles: ExpectedRoles,
		l:             logger.New("auth_server", color.New(color.FgHiRed)),
	}
}

// PokerAuthFunc is used by a middleware to authenticate requests
func (s *Server) PokerAuthFunc(ctx context.Context) (context.Context, error) {
	tokenStr, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}

	if _, err = s.validateToken(ctx, tokenStr, Realm); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	uinfo, err := s.cloakClient.GetUserInfo(ctx, tokenStr, Realm)
	if err != nil {
		spew.Dump(uinfo)
		return nil, err
	}
	grpc_ctxtags.Extract(ctx).Set("auth.sub", uinfo.Sub)
	newCtx := context.WithValue(ctx, UinfoType("uinfo"), uinfo)

	return newCtx, nil
}

// UinfoType is the userInfo type for keycloak
type UinfoType string

func (s *Server) validateRoles(claims jwt.MapClaims) error {
	var resourceAccess map[string]interface{}
	var clientMap map[string]interface{}
	var roles []interface{}
	var ok bool

	if resourceAccess, ok = claims["resource_access"].(map[string]interface{}); !ok {
		return fmt.Errorf("resource_access key missing")
	}

	if clientMap, ok = resourceAccess[ClientID].(map[string]interface{}); !ok {
		return fmt.Errorf("client key missing")
	}

	if roles, ok = clientMap["roles"].([]interface{}); !ok {
		return fmt.Errorf("roles key missing")
	}

	var myRoles []string
	for _, role := range roles {
		myRoles = append(myRoles, role.(string))
	}

	if !haveRoles(s.ExpectedRoles, myRoles) {
		return fmt.Errorf("missing required roles: %v", s.ExpectedRoles)
	}

	return nil
}

func (s *Server) validateToken(ctx context.Context, token, realm string) (*jwt.Token, error) {

	// This calls out to login.wetsnow.com for cert info
	t, claims, err := s.cloakClient.DecodeAccessToken(ctx, token, realm, Audience)
	if err != nil {
		return nil, err
	}

	// This calls out to the server
	s.l.Debug("retrospecting token...")
	if os.Getenv(clientSecretENV) == "" {
		s.l.Fatalf("Please set the [%v] environment variable...", clientSecretENV)
	}
	res, err := s.cloakClient.RetrospectToken(ctx, token, "pepper-poker.wetsnow.com", os.Getenv(clientSecretENV), realm)
	if err != nil {
		return nil, err
	}
	if !*res.Active {
		return nil, fmt.Errorf("provided access token not valid")
	}
	s.l.Debug(spew.Sdump(res))

	vHelper := jwt.NewValidationHelper(jwt.WithAudience(Audience), jwt.WithIssuer(Issuer))

	// offline validation (this does almost no validation by default)
	if err := claims.Valid(vHelper); err != nil {
		s.l.Error(err)
		s.l.Error(spew.Sdump(claims))
		return nil, err
	}

	if err := s.validateRoles(*claims); err != nil {
		return nil, err
	}

	return t, nil
}

// Client is the auth module for the client
type Client struct {
	l *logger.Logger
}

// NewClient returns a new auth client
func NewClient() *Client {
	return &Client{

		l: logger.New("auth_client", color.New(color.FgHiRed)),
	}
}

// oAuthClientConfig returns the oauth client config for retrieving tokens
func (c *Client) oAuthClientConfig() *oauth2.Config {
	conf := &oauth2.Config{
		ClientID: ClientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  AuthURL,
			TokenURL: TokenURL,
		},
	}
	return conf
}

// GetAuthToken gets an auth token based on the username and password of the client
func (c *Client) GetAuthToken(ctx context.Context, username, password string) (*oauth2.Token, error) {
	conf := c.oAuthClientConfig()
	return conf.PasswordCredentialsToken(ctx, username, password)
}

func roleInList(role string, roles []string) bool {
	for _, r := range roles {
		if role == r {
			return true
		}
	}
	return false
}

func haveRoles(expected, have []string) bool {
	for _, role := range expected {
		if !roleInList(role, have) {
			return false
		}
	}
	return true
}
