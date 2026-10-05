package oidcverifier

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

func TestVerifierValidatesSupabaseStyleOIDCToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := jose.JSONWebKey{
		Key:       &privateKey.PublicKey,
		KeyID:     "test-key",
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
	jwks, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{publicKey}})
	if err != nil {
		t.Fatal(err)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"issuer":                                server.URL,
				"authorization_endpoint":                server.URL + "/authorize",
				"token_endpoint":                        server.URL + "/token",
				"jwks_uri":                              server.URL + "/.well-known/jwks.json",
				"response_types_supported":              []string{"code"},
				"subject_types_supported":               []string{"public"},
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/.well-known/jwks.json":
			_, _ = writer.Write(jwks)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	verifier, err := NewVerifier(context.Background(), server.URL, "authenticated")
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}

	rawToken := signedToken(t, privateKey, map[string]any{
		"iss":            server.URL,
		"sub":            "supabase-user-id",
		"aud":            "authenticated",
		"exp":            time.Now().Add(time.Hour).Unix(),
		"iat":            time.Now().Unix(),
		"email":          "active@example.com",
		"email_verified": true,
		"app_metadata": map[string]any{
			"roles": []string{"super_admin"},
		},
	})

	claims, err := verifier.Verify(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if claims.Subject != "supabase-user-id" || claims.Email != "active@example.com" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if len(claims.AppMetadata.Roles) != 1 || claims.AppMetadata.Roles[0] != "super_admin" {
		t.Fatalf("unexpected app metadata roles: %+v", claims.AppMetadata.Roles)
	}
}

func TestVerifierRejectsExpiredAndUnsupportedAlgorithmTokens(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := jose.JSONWebKey{Key: &privateKey.PublicKey, KeyID: "test-key", Algorithm: string(jose.RS256), Use: "sig"}
	jwks, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{publicKey}})
	if err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"issuer": server.URL, "jwks_uri": server.URL + "/.well-known/jwks.json",
				"response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"},
				"id_token_signing_alg_values_supported": []string{"RS256", "HS256", "none"},
			})
		case "/.well-known/jwks.json":
			_, _ = writer.Write(jwks)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	verifier, err := NewVerifier(context.Background(), server.URL, "authenticated")
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}

	expired := signedToken(t, privateKey, map[string]any{
		"iss": server.URL, "sub": "subject", "aud": "authenticated",
		"exp": time.Now().Add(-time.Minute).Unix(), "iat": time.Now().Add(-time.Hour).Unix(),
	})
	if _, err := verifier.Verify(context.Background(), expired); err == nil {
		t.Fatal("Verify() accepted an expired token")
	}
	notYetValid := signedToken(t, privateKey, map[string]any{
		"iss": server.URL, "sub": "subject", "aud": "authenticated",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
		"nbf": time.Now().Add(time.Minute).Unix(),
	})
	if _, err := verifier.Verify(context.Background(), notYetValid); err == nil {
		t.Fatal("Verify() accepted a token outside the configured not-before leeway")
	}

	payload, err := json.Marshal(map[string]any{
		"iss": server.URL, "sub": "subject", "aud": "authenticated",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	hmacSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: []byte("01234567890123456789012345678901")}, (&jose.SignerOptions{}).WithHeader("kid", "test-key"))
	if err != nil {
		t.Fatal(err)
	}
	hmacJWT, err := hmacSigner.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	hmacToken, err := hmacJWT.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(context.Background(), hmacToken); err == nil {
		t.Fatal("Verify() accepted an HS256 token")
	}
	if _, err := verifier.Verify(context.Background(), "eyJhbGciOiJub25lIn0.eyJzdWIiOiJzdWJqZWN0In0."); err == nil {
		t.Fatal("Verify() accepted a token with alg=none")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestPacedRoundTripperLimitsUnknownKeyRefreshes(t *testing.T) {
	transport := &pacedRoundTripper{
		base: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		}),
		interval: 50 * time.Millisecond,
	}
	start := time.Now()
	for range 3 {
		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://issuer.example/jwks", nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := transport.RoundTrip(request); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed < 90*time.Millisecond {
		t.Fatalf("three key fetches completed in %s; expected rate limiting", elapsed)
	}
}

func TestVerifierRejectsWrongAudience(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := jose.JSONWebKey{
		Key:       &privateKey.PublicKey,
		KeyID:     "test-key",
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
	jwks, err := json.Marshal(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{publicKey}})
	if err != nil {
		t.Fatal(err)
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"issuer":                                server.URL,
				"jwks_uri":                              server.URL + "/.well-known/jwks.json",
				"response_types_supported":              []string{"code"},
				"subject_types_supported":               []string{"public"},
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/.well-known/jwks.json":
			_, _ = writer.Write(jwks)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	verifier, err := NewVerifier(context.Background(), server.URL, "authenticated")
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	rawToken := signedToken(t, privateKey, map[string]any{
		"iss": server.URL,
		"sub": "supabase-user-id",
		"aud": "other",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	if _, err := verifier.Verify(context.Background(), rawToken); err == nil {
		t.Fatal("Verify() accepted a token with the wrong audience")
	}
}

func signedToken(t *testing.T, privateKey *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()

	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: privateKey},
		(&jose.SignerOptions{}).WithHeader("kid", "test-key"),
	)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}
	token, err := signed.CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}
