package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/genesis32/complianceweb/auth"

	"github.com/genesis32/complianceweb/utils"

	"github.com/coreos/go-oidc"
	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

// IndexHandler is just a placeholder for now.
func IndexHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Welcome",
	})
	return nil
}

func InviteHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {
	if c.Request.Method == "GET" {
		inviteCodeStr := c.Param("inviteCode")
		inviteCode, _ := utils.StringToInt64(inviteCodeStr)
		theUser := daoHandler.LoadUserFromInviteCode(inviteCode)
		if theUser == nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("invite code not valid"))
			return nil
		}
		href := createInviteLink("", inviteCode, daoHandler)
		c.Redirect(302, href)
	}
	return nil
}

// LoginHandler initiate the login flow.
func LoginHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, dao dao.DaoHandler, c *gin.Context) *WebAppOperationResult {
	var auth0Authenticator *auth.Auth0Authenticator
	var ok bool
	if auth0Authenticator, ok = s.Authenticator.(*auth.Auth0Authenticator); !ok {
		log.Fatalf("Webforms only support Auth0Authenticator")
	}

	w := c.Writer
	r := c.Request
	// Generate random state
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}
	state := base64.StdEncoding.EncodeToString(b)

	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	// TODO: Hack to get an invite code into the callback
	inviteCode := c.Query("inviteCode")
	if inviteCode != "" {
		state += fmt.Sprintf("|%s", inviteCode)
	}

	session.Values["state"] = state
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	http.Redirect(w, r, auth0Authenticator.Config.AuthCodeURL(state), http.StatusTemporaryRedirect)
	return nil
}

// CallbackHandler handles the redirect from auth0.
func CallbackHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, dao dao.DaoHandler, c *gin.Context) *WebAppOperationResult {
	var auth0Authenticator *auth.Auth0Authenticator
	var ok bool
	if auth0Authenticator, ok = s.Authenticator.(*auth.Auth0Authenticator); !ok {
		log.Fatalf("Webforms only support Auth0Authenticator")
	}

	w := c.Writer
	r := c.Request

	settings := dao.GetSettings(Auth0ClientIDConfigurationKey)
	if len(settings) == 0 {
		log.Fatal("no clientid configured")
	}

	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	if r.URL.Query().Get("state") != session.Values["state"] {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return nil
	}

	token, err := auth0Authenticator.Config.Exchange(context.TODO(), r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("no token found: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
		return nil
	}

	oidcConfig := &oidc.Config{
		ClientID: settings[Auth0ClientIDConfigurationKey].Value,
	}

	idToken, err := auth0Authenticator.Provider.Verifier(oidcConfig).Verify(context.TODO(), rawIDToken)

	if err != nil {
		http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	// Getting now the userInfo
	var profile map[string]interface{}
	if err := idToken.Claims(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	stateWithInvite := strings.Split(r.URL.Query().Get("state"), "|")
	if len(stateWithInvite) > 1 {
		initialized := dao.InitUserFromInviteCode(stateWithInvite[1], fmt.Sprintf("%v", profile["sub"]))
		if !initialized {
			http.Error(w, "Failed to initialize user", http.StatusOK)
			return nil
		}
	}

	organizationUser, err := dao.LogUserIn(profile["sub"].(string))
	if organizationUser == nil && err == nil {
		http.Redirect(w, r, "/webapp", http.StatusSeeOther)
		return nil
	}

	if err != nil {
		http.Error(w, "Failed to initialize user: "+err.Error(), http.StatusInternalServerError)
		return nil
	}

	session.Values["id_token"] = rawIDToken
	session.Values["access_token"] = token.AccessToken
	session.Values["profile"] = profile
	session.Values["organization_user"] = organizationUser
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	c.JSON(200, gin.H{
		"idToken": rawIDToken,
	})

	return nil
}
