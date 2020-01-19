package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"golang.org/x/oauth2"

	oidc "github.com/coreos/go-oidc"
	"github.com/dgrijalva/jwt-go"
)

type OpenIDClaims map[string]interface{}

type Authenticator interface {
	ValidateAuthorizationHeader(headerValue string) (OpenIDClaims, error)
}

type TestAuthenticator struct {
}

type Auth0Authenticator struct {
	Provider *oidc.Provider
	Config   oauth2.Config
	Ctx      context.Context
	verifier *oidc.IDTokenVerifier
}

var bearerRegex = regexp.MustCompile("[B|b]earer\\s+(\\S+)")

func (a *TestAuthenticator) ValidateAuthorizationHeader(headerValue string) (OpenIDClaims, error) {
	log.Printf("proccesing token with TestAuthenticator")

	hv := strings.TrimSpace(headerValue)

	if hv == "" {
		return nil, errors.New("headerValue is blank")
	}

	rs := bearerRegex.FindStringSubmatch(hv)
	if rs == nil || len(rs) < 2 {
		return nil, errors.New("cannot parse header")
	}

	hmacSecret := make([]byte, 64)
	token, err := jwt.Parse(rs[1], func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return hmacSecret, nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		openIDClaims := make(OpenIDClaims)
		for k, v := range claims {
			openIDClaims[k] = v
		}
		return openIDClaims, nil
	} else {
		return nil, err
	}
}

func NewTestAuthenticator() Authenticator {
	log.Printf("WARNING USING A TEST AUTHENTICATOR")
	return &TestAuthenticator{}
}

func (a *Auth0Authenticator) ValidateAuthorizationHeader(headerValue string) (OpenIDClaims, error) {

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

func NewAuth0Authenticator(callbackUrl, issuerBaseUrl, auth0ClientID, auth0ClientSecret string) Authenticator {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, issuerBaseUrl)
	if err != nil {
		log.Fatalf("Failed to get provider: %v", err)
	}

	conf := oauth2.Config{
		ClientID:     auth0ClientID,
		ClientSecret: auth0ClientSecret,
		RedirectURL:  callbackUrl,
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
