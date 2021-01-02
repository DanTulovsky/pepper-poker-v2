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
	cloakClient = gocloak.NewClient(oidcProviderURL)
)

const (
	// TODO: combine with client
	oidcProviderURL = "https://login.wetsnow.com"
	realm           = "wetsnow"
	audience        = "pepper-poker-grpc"
)

type uinfoType string

func validateToken(ctx context.Context, token, realm string) (*jwt.Token, error) {

	t, claims, err := cloakClient.DecodeAccessToken(ctx, token, realm, audience)
	if err != nil {
		return nil, err
	}

	log.Printf("%#v", t)
	log.Printf("%#v", claims)

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

	return t, nil
}

func userClaimFromToken(struct{}) string {
	return "foobar"
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

	return newCtx, nil
}
