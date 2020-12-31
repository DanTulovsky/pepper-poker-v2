package server

import (
	"context"
	"fmt"
	"log"

	gocloak "github.com/Nerzal/gocloak/v7"
	"github.com/davecgh/go-spew/spew"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	// TODO: combine with client
	oidcProviderURL = "https://login.wetsnow.com"
	realm           = "wetsnow"
	audience        = "pepper-poker-grpc"
)

func parseToken(ctx context.Context, token, realm string) (struct{}, error) {
	client := gocloak.NewClient(oidcProviderURL)

	t, claims, err := client.DecodeAccessToken(ctx, token, realm, audience)
	if err != nil {
		return struct{}{}, err
	}

	log.Printf("%#v", t)
	log.Printf("%#v", claims)

	uinfo, err := client.GetUserInfo(ctx, token, realm)
	if err != nil {
		return struct{}{}, err
	}
	spew.Dump(uinfo)

	return struct{}{}, nil
}

func userClaimFromToken(struct{}) string {
	return "foobar"
}

// pokerAuthFunc is used by a middleware to authenticate requests
func pokerAuthFunc(ctx context.Context) (context.Context, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, err
	}

	m, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("error retrieving metadata from context")
	}

	log.Printf("%#v", m)

	tokenInfo, err := parseToken(ctx, token, realm)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	grpc_ctxtags.Extract(ctx).Set("auth.sub", userClaimFromToken(tokenInfo))

	newCtx := context.WithValue(ctx, "tokenInfo", tokenInfo)

	return newCtx, nil
}
