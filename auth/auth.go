package auth

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"

	"golang.org/x/oauth2"

	oidc "github.com/coreos/go-oidc"
)

type Authenticator struct {
	Provider *oidc.Provider
	Config   oauth2.Config
	Ctx      context.Context
	verifier *oidc.IDTokenVerifier
}

type OpenIDClaims map[string]interface{}

var bearerRegex = regexp.MustCompile("[B|b]earer\\s+(\\S+)")

func (a *Authenticator) ValidateAuthorizationHeader(headerValue string) (OpenIDClaims, error) {

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

func NewAuthenticator(issuerBaseUrl, auth0ClientID, auth0ClientSecret string) (*Authenticator, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, issuerBaseUrl)
	if err != nil {
		log.Printf("failed to get provider: %v", err)
		return nil, err
	}

	conf := oauth2.Config{
		ClientID:     auth0ClientID,
		ClientSecret: auth0ClientSecret,
		RedirectURL:  "http://localhost:3000/webapp/callback",
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}

	oidcConfig := &oidc.Config{
		ClientID:             auth0ClientID,
		SupportedSigningAlgs: []string{"RS256"},
	}

	theVerifier := provider.Verifier(oidcConfig)

	return &Authenticator{
		Provider: provider,
		Config:   conf,
		Ctx:      ctx,
		verifier: theVerifier,
	}, nil
}
