package auth

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"

	"github.com/genesis32/complianceweb/utils"

	"golang.org/x/oauth2"

	oidc "github.com/coreos/go-oidc"
)

// Authenticator provies the interface to validate a bearer token and return user claims
type Authenticator interface {
	ValidateAuthorizationHeader(headerValue string) (utils.OpenIDClaims, error)
}

// TestAuthenticator just validates a jwt
type TestAuthenticator struct {
}

// Auth0Authenticator validates a jwt from auth0
type Auth0Authenticator struct {
	Provider *oidc.Provider
	Config   oauth2.Config
	Ctx      context.Context
	verifier *oidc.IDTokenVerifier
}

var bearerRegex = regexp.MustCompile("[B|b]earer\\s+(\\S+)")

// ValidateAuthorizationHeader validates the simple jwt
func (a *TestAuthenticator) ValidateAuthorizationHeader(headerValue string) (utils.OpenIDClaims, error) {

	hv := strings.TrimSpace(headerValue)

	if hv == "" {
		return nil, errors.New("headerValue is blank")
	}

	rs := bearerRegex.FindStringSubmatch(hv)
	if rs == nil || len(rs) < 2 {
		return nil, errors.New("cannot parse header")
	}

	hmacSecret := make([]byte, 64)
	claims := utils.ParseTestJwt(rs[1], hmacSecret)
	return claims, nil
}

// NewTestAuthenticator returns a new jwt authenticator
func NewTestAuthenticator() Authenticator {
	log.Printf("WARNING USING A TEST AUTHENTICATOR THAT ONLY SUPPORTS HS256")
	return &TestAuthenticator{}
}

// ValidateAuthorizationHeader validates and auth0 simple jwt
func (a *Auth0Authenticator) ValidateAuthorizationHeader(headerValue string) (utils.OpenIDClaims, error) {

	hv := strings.TrimSpace(headerValue)

	if hv == "" {
		return nil, errors.New("headerValue is blank")
	}

	rs := bearerRegex.FindStringSubmatch(hv)
	if rs == nil || len(rs) < 2 {
		return nil, errors.New("cannot parse header")
	}

	// TODO: Check nonce
	idToken, err := a.verifier.Verify(context.TODO(), rs[1])

	if err != nil {
		return nil, err
	}

	// Getting now the userInfo
	var profile map[string]interface{}
	if err := idToken.Claims(&profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// NewAuth0Authenticator returns new auth0 authenticator.
func NewAuth0Authenticator(callbackURL, issuerBaseURL, auth0ClientID, auth0ClientSecret string) Authenticator {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, issuerBaseURL)
	if err != nil {
		log.Fatalf("Failed to get provider: %v", err)
	}

	conf := oauth2.Config{
		ClientID:     auth0ClientID,
		ClientSecret: auth0ClientSecret,
		RedirectURL:  callbackURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}

	oidcConfig := &oidc.Config{
		ClientID:             auth0ClientID,
		SupportedSigningAlgs: []string{"RS256"},
	}

	theVerifier := provider.Verifier(oidcConfig)

	return &Auth0Authenticator{
		Provider: provider,
		Config:   conf,
		Ctx:      ctx,
		verifier: theVerifier,
	}
}
