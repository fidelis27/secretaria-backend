package oidcverifier

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

func NewVerifier(ctx context.Context, issuer, audience string) (*Verifier, error) {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	audience = strings.TrimSpace(audience)
	if issuer == "" || audience == "" {
		return nil, fmt.Errorf("OIDC_ISSUER and OIDC_AUDIENCE are required")
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC issuer: %w", err)
	}
	return &Verifier{verifier: provider.Verifier(&oidc.Config{ClientID: audience})}, nil
}

func (verifier *Verifier) Verify(ctx context.Context, rawToken string) (httpserver.IdentityClaims, error) {
	token, err := verifier.verifier.Verify(ctx, rawToken)
	if err != nil {
		return httpserver.IdentityClaims{}, fmt.Errorf("verify OIDC token: %w", err)
	}
	var claims httpserver.IdentityClaims
	if err := token.Claims(&claims); err != nil {
		return httpserver.IdentityClaims{}, fmt.Errorf("decode OIDC claims: %w", err)
	}
	return claims, nil
}
