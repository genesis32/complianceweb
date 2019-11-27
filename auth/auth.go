package auth

import (
	"context"
	"errors"
	"log"
	"os"
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

	idToken, err := a.verifier.Verify(context.TODO(), rs[1])

	if err != nil {
		//		http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
		return nil, err
	}

	// Getting now the userInfo
	var profile map[string]interface{}
	if err := idToken.Claims(&profile); err != nil {
		//		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}
	return profile, nil
}

func NewAuthenticator() (*Authenticator, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, "https://***REMOVED***.auth0.com/")
	if err != nil {
		log.Printf("failed to get provider: %v", err)
		return nil, err
	}

	clientId := os.Getenv("AUTH0_CLIENT_ID")
	if len(clientId) == 0 {
		panic("AUTH0_CLIENT_ID undefined")
	}

	clientSecret := os.Getenv("AUTH0_CLIENT_SECRET")
	if len(clientSecret) == 0 {
		panic("AUTH0_CLIENT_SECRET undefined")
	}

	conf := oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:3000/webapp/callback",
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}

	oidcConfig := &oidc.Config{
		ClientID: "***REMOVED***",
	}

	theVerifier := provider.Verifier(oidcConfig)

	return &Authenticator{
		Provider: provider,
		Config:   conf,
		Ctx:      ctx,
		verifier: theVerifier,
	}, nil
}
