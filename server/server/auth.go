package server

import (
	"context"
	"fmt"
	"log"
	"os"

	gocloak "github.com/Nerzal/gocloak/v7"
	"github.com/davecgh/go-spew/spew"
	"github.com/dgrijalva/jwt-go/v4"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DanTulovsky/pepper-poker-v2/auth"
)

var (
	cloakClient   = gocloak.NewClient(auth.OIDCProviderURL)
	expectedRoles = []string{"user"}
)

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

type uinfoType string

func validateRoles(claims jwt.MapClaims) error {
	var resourceAccess map[string]interface{}
	var ok bool

	if resourceAccess, ok = claims["resource_access"].(map[string]interface{}); !ok {
		return fmt.Errorf("resource_access key missing")
	}

	for key, value := range resourceAccess {
		log.Printf("%#v: %#v", key, value)
	}

	var clientMap map[string]interface{}

	if clientMap, ok = resourceAccess[auth.ClientID].(map[string]interface{}); !ok {
		return fmt.Errorf("client key missing")
	}

	for key, value := range clientMap {
		log.Printf("%#v (%T): %#v", key, key, value)
	}

	var roles []interface{}
	if roles, ok = clientMap["roles"].([]interface{}); !ok {
		return fmt.Errorf("roles key missing")
	}

	var myRoles []string
	for _, role := range roles {
		log.Printf("> %v", role.(string))
		myRoles = append(myRoles, role.(string))
	}

	if !haveRoles(expectedRoles, myRoles) {
		return fmt.Errorf("missing required roles: %v", expectedRoles)
	}
	return nil
}

func validateToken(ctx context.Context, token, realm string) (*jwt.Token, error) {

	// This calls out to login.wetsnow.com for cert info
	t, claims, err := cloakClient.DecodeAccessToken(ctx, token, realm, auth.Audience)
	if err != nil {
		return nil, err
	}

	log.Printf("%#v", t)
	log.Printf("%#v", claims)

	// This calls out to the server
	log.Printf("retrospecting token...")
	res, err := cloakClient.RetrospectToken(ctx, token, "pepper-poker.wetsnow.com", os.Getenv("PEPPER_POKER_CLIENT_SECRET"), "wetsnow")
	if err != nil {
		log.Printf("error retrospecting: %v", err)
		return nil, err
	}
	if !*res.Active {
		return nil, fmt.Errorf("provided access token not valid")
	}
	log.Print("Introspection response...")
	spew.Dump(res)

	vHelper := jwt.NewValidationHelper(jwt.WithAudience(auth.Audience), jwt.WithIssuer(auth.Issuer))

	// offline validation (this does almost no validation by default)
	if err := claims.Valid(vHelper); err != nil {
		log.Print(err)
		spew.Dump(claims)
		return nil, err
	}

	if err := validateRoles(*claims); err != nil {
		return nil, err
	}

	return t, nil
}

// pokerAuthFunc is used by a middleware to authenticate requests
func pokerAuthFunc(ctx context.Context) (context.Context, error) {
	tokenStr, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}

	if _, err = validateToken(ctx, tokenStr, auth.Realm); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	uinfo, err := cloakClient.GetUserInfo(ctx, tokenStr, auth.Realm)
	if err != nil {
		spew.Dump(uinfo)
		return nil, err
	}
	grpc_ctxtags.Extract(ctx).Set("auth.sub", uinfo.Sub)

	newCtx := context.WithValue(ctx, uinfoType("uinfo"), uinfo)

	return newCtx, nil
}
