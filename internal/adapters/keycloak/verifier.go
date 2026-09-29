package keycloak

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/fidelis27/secretaria-backend/internal/adapters/httpserver"
)

type Verifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewVerifier(ctx context.Context, issuer, clientID string) (*Verifier, error) {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	clientID = strings.TrimSpace(clientID)
	if issuer == "" || clientID == "" {
		return nil, fmt.Errorf("KEYCLOAK_ISSUER and KEYCLOAK_CLIENT_ID are required")
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover Keycloak issuer: %w", err)
	}
	return &Verifier{verifier: provider.Verifier(&oidc.Config{ClientID: clientID})}, nil
}

func (verifier *Verifier) Verify(ctx context.Context, rawToken string) (httpserver.IdentityClaims, error) {
	token, err := verifier.verifier.Verify(ctx, rawToken)
	if err != nil {
		return httpserver.IdentityClaims{}, fmt.Errorf("verify Keycloak token: %w", err)
	}
	var claims httpserver.IdentityClaims
	if err := token.Claims(&claims); err != nil {
		return httpserver.IdentityClaims{}, fmt.Errorf("decode Keycloak claims: %w", err)
	}
	return claims, nil
}
