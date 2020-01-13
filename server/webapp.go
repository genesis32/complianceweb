package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/genesis32/complianceweb/utils"

	"github.com/coreos/go-oidc"
	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

func IndexHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Welcome",
	})
}

func InviteHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	if c.Request.Method == "GET" {
		inviteCodeStr := c.Param("inviteCode")
		inviteCode, _ := utils.StringToInt64(inviteCodeStr)
		theUser, err := daoHandler.LoadUserFromInviteCode(inviteCode)
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("error getting invite code: %s", err.Error()))
			return
		}
		if theUser == nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("invite code not valid"))
			return
		}
		href := createInviteLink("", inviteCode, daoHandler)
		c.Redirect(302, href)
	}
}

func LoginHandler(s *Server, store sessions.Store, dao dao.DaoHandler, c *gin.Context) {
	w := c.Writer
	r := c.Request
	// Generate random state
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	state := base64.StdEncoding.EncodeToString(b)

	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		return
	}

	http.Redirect(w, r, s.Authenticator.Config.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func CallbackHandler(s *Server, store sessions.Store, dao dao.DaoHandler, c *gin.Context) {
	w := c.Writer
	r := c.Request

	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.URL.Query().Get("state") != session.Values["state"] {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	token, err := s.Authenticator.Config.Exchange(context.TODO(), r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("no token found: %v", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
		return
	}

	oidcConfig := &oidc.Config{
		ClientID: "***REMOVED***",
	}

	idToken, err := s.Authenticator.Provider.Verifier(oidcConfig).Verify(context.TODO(), rawIDToken)

	if err != nil {
		http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Getting now the userInfo
	var profile map[string]interface{}
	if err := idToken.Claims(&profile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	stateWithInvite := strings.Split(r.URL.Query().Get("state"), "|")
	if len(stateWithInvite) > 1 {
		initialized, err := dao.InitUserFromInviteCode(stateWithInvite[1], fmt.Sprintf("%v", profile["sub"]))
		if !initialized {
			http.Error(w, "Failed to initialize user", http.StatusOK)
			return
		}
		if err != nil {
			http.Error(w, "Failed to initialize user: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	organizationUser, err := dao.LogUserIn(profile["sub"].(string))
	if organizationUser == nil && err == nil {
		http.Redirect(w, r, "/webapp", http.StatusSeeOther)
		return
	}

	if err != nil {
		http.Error(w, "Failed to initialize user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values["id_token"] = rawIDToken
	session.Values["access_token"] = token.AccessToken
	session.Values["profile"] = profile
	session.Values["organization_user"] = organizationUser
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c.JSON(200, gin.H{
		"idToken": rawIDToken,
	})

}