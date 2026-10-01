package httpserver

import (
	"context"
	"net/http"
	"strings"

	applicationuser "github.com/fidelis27/secretaria-backend/internal/application/user"
	domainuser "github.com/fidelis27/secretaria-backend/internal/domain/user"
)

type identityContextKey struct{}

type identityClaimsContextKey struct{}

type IdentityClaims struct {
	Subject       string   `json:"sub"`
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Roles         []string `json:"roles"`
	AppMetadata   struct {
		Roles []string `json:"roles"`
	} `json:"app_metadata"`
}

type identityTokenVerifier interface {
	Verify(context.Context, string) (IdentityClaims, error)
}

func identityMiddleware(next http.Handler, users applicationuser.Service, verifier identityTokenVerifier) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/health" {
			next.ServeHTTP(writer, request)
			return
		}

		token := bearerToken(request)
		if token == "" {
			writeError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}
		claims, err := verifier.Verify(request.Context(), token)
		if err != nil || strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(claims.Email) == "" || !claims.EmailVerified {
			writeError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}
		user, found, err := users.FindByEmail(request.Context(), strings.TrimSpace(claims.Email))
		if err != nil {
			writeError(writer, http.StatusInternalServerError, "could not load user")
			return
		}
		if !found || user.Status != "active" {
			writeError(writer, http.StatusForbidden, "forbidden")
			return
		}
		user.SuperAdmin = user.SuperAdmin && hasIdentityRole(claims, "super_admin")

		ctx := context.WithValue(request.Context(), identityContextKey{}, user)
		ctx = context.WithValue(ctx, identityClaimsContextKey{}, claims)
		request = request.WithContext(ctx)
		next.ServeHTTP(writer, request)
	})
}

func bearerToken(request *http.Request) string {
	if header := strings.TrimSpace(request.Header.Get("Authorization")); header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	if websocketRequest(request) {
		for _, protocol := range strings.Split(request.Header.Get("Sec-WebSocket-Protocol"), ",") {
			protocol = strings.TrimSpace(protocol)
			if strings.HasPrefix(protocol, "bearer.") {
				return strings.TrimPrefix(protocol, "bearer.")
			}
		}
	}
	return ""
}

func hasRole(roles []string, expected string) bool {
	for _, role := range roles {
		if role == expected {
			return true
		}
	}
	return false
}

func hasIdentityRole(claims IdentityClaims, expected string) bool {
	return hasRole(claims.Roles, expected) ||
		hasRole(claims.AppMetadata.Roles, expected)
}

func websocketRequest(request *http.Request) bool {
	return strings.EqualFold(request.Header.Get("Upgrade"), "websocket")
}

func userFromContext(ctx context.Context) (domainuser.User, bool) {
	user, ok := ctx.Value(identityContextKey{}).(domainuser.User)
	return user, ok
}

func identityClaimsFromContext(ctx context.Context) (IdentityClaims, bool) {
	claims, ok := ctx.Value(identityClaimsContextKey{}).(IdentityClaims)
	return claims, ok
}
