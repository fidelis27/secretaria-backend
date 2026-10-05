package oidcverifier

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/fidelis27/secretaria-backend/internal/adapters/httpserver"
)

type Verifier struct {
	verifier *oidc.IDTokenVerifier
}

const oidcClockLeeway = 30 * time.Second

func NewVerifier(ctx context.Context, issuer, audience string) (*Verifier, error) {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	audience = strings.TrimSpace(audience)
	if issuer == "" || audience == "" {
		return nil, fmt.Errorf("OIDC_ISSUER and OIDC_AUDIENCE are required")
	}
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &pacedRoundTripper{base: http.DefaultTransport, interval: time.Second},
	}
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, client), issuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC issuer: %w", err)
	}
	return &Verifier{verifier: provider.Verifier(&oidc.Config{
		ClientID:             audience,
		SupportedSigningAlgs: []string{"RS256", "ES256"},
		Now:                  func() time.Time { return time.Now().Add(-oidcClockLeeway) },
	})}, nil
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
	var temporalClaims struct {
		NotBefore *int64 `json:"nbf"`
	}
	if err := token.Claims(&temporalClaims); err != nil {
		return httpserver.IdentityClaims{}, fmt.Errorf("decode OIDC temporal claims: %w", err)
	}
	now := time.Now()
	if token.Expiry.Before(now.Add(-oidcClockLeeway)) {
		return httpserver.IdentityClaims{}, fmt.Errorf("verify OIDC token: token expired")
	}
	if temporalClaims.NotBefore != nil && time.Unix(*temporalClaims.NotBefore, 0).After(now.Add(oidcClockLeeway)) {
		return httpserver.IdentityClaims{}, fmt.Errorf("verify OIDC token: token is not yet valid")
	}
	if token.IssuedAt.After(now.Add(oidcClockLeeway)) {
		return httpserver.IdentityClaims{}, fmt.Errorf("verify OIDC token: issued-at time is in the future")
	}
	return claims, nil
}

type pacedRoundTripper struct {
	base     http.RoundTripper
	interval time.Duration
	mutex    sync.Mutex
	next     time.Time
}

func (transport *pacedRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.mutex.Lock()
	now := time.Now()
	start := now
	if transport.next.After(start) {
		start = transport.next
	}
	transport.next = start.Add(transport.interval)
	transport.mutex.Unlock()

	if delay := time.Until(start); delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-request.Context().Done():
			return nil, request.Context().Err()
		case <-timer.C:
		}
	}
	return transport.base.RoundTrip(request)
}
