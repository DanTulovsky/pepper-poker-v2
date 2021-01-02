package server

import (
	"context"
	"fmt"
	"log"

	gocloak "github.com/Nerzal/gocloak/v7"
	"github.com/davecgh/go-spew/spew"
	"github.com/dgrijalva/jwt-go/v4"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	cloakClient   = gocloak.NewClient(oidcProviderURL)
	expectedRoles = []string{"user"}
)

const (
	// TODO: combine with client
	oidcProviderURL = "https://login.wetsnow.com"
	realm           = "wetsnow"
	audience        = "pepper-poker-grpc"
	client          = "pepper-poker-grpc.wetsnow.com"
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

	if clientMap, ok = resourceAccess[client].(map[string]interface{}); !ok {
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

	t, claims, err := cloakClient.DecodeAccessToken(ctx, token, realm, audience)
	if err != nil {
		return nil, err
	}

	log.Printf("%#v", t)
	log.Printf("%#v", claims)

	// This calls out to the server
	log.Printf("retrospecting token...")
	res, err := cloakClient.RetrospectToken(ctx, token, "pepper-poker.wetsnow.com", "b24e6370-2c12-44d2-85db-43fb79ab3382", "wetsnow")
	if err != nil {
		log.Printf("error retrospecting: %v", err)
		return nil, err
	}
	if !*res.Active {
		spew.Dump(res)
		return nil, fmt.Errorf("provided access token not valid")
	}

	vHelper := jwt.NewValidationHelper(jwt.WithAudience(audience))

	// offline validation (this does almost no validation by default)
	if err := claims.Valid(vHelper); err != nil {
		log.Print(err)
		spew.Dump(claims)
		return nil, err
	}

	// TODO: Create custom validator to check roles?
	for key, value := range *claims {
		log.Printf("%v: %v", key, value)

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

	// m, ok := metadata.FromIncomingContext(ctx)
	// if !ok {
	// 	return nil, fmt.Errorf("error retrieving metadata from context")
	// }

	// log.Printf("%#v", m)

	if _, err = validateToken(ctx, tokenStr, realm); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	uinfo, err := cloakClient.GetUserInfo(ctx, tokenStr, realm)
	if err != nil {
		spew.Dump(uinfo)
		return nil, err
	}
	grpc_ctxtags.Extract(ctx).Set("auth.sub", uinfo.Sub)

	newCtx := context.WithValue(ctx, uinfoType("uinfo"), uinfo)

	// claims := userClaimFromToken(token)

	return newCtx, nil
}
