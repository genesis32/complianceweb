package webhandlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/coreos/go-oidc"
	"github.com/genesis32/complianceweb/auth"
	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
)

func IndexHandler(store sessions.Store, dao dao.DaoHandler, c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Welcome",
	})
}

func ProfileHandler(store sessions.Store, dao dao.DaoHandler, c *gin.Context) {
	session, err := store.Get(c.Request, "auth-session")
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	v, ok := session.Values["profile"].(map[string]interface{})
	if !ok {
		c.HTML(http.StatusInternalServerError, "index.tmpl", gin.H{
			"title": "error",
		})
		return
	}

	dao.CreateOrUpdateUser(fmt.Sprintf("%v", v["name"]), fmt.Sprintf("%v", v["sub"]))

	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": fmt.Sprintf("User: %+v", session.Values["profile"]),
	})
}

func CallbackHandler(store sessions.Store, dao dao.DaoHandler, c *gin.Context) {
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

	authenticator, err := auth.NewAuthenticator()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := authenticator.Config.Exchange(context.TODO(), r.URL.Query().Get("code"))
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

	idToken, err := authenticator.Provider.Verifier(oidcConfig).Verify(context.TODO(), rawIDToken)

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

	session.Values["id_token"] = rawIDToken
	session.Values["access_token"] = token.AccessToken
	session.Values["profile"] = profile
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to logged in page
	http.Redirect(w, r, "/webapp/profile", http.StatusSeeOther)
}

func BootstrapHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	if c.Request.Method == "GET" {
		c.HTML(http.StatusOK, "createOrg.tmpl", gin.H{
			"title":          "Create Org",
			csrf.TemplateTag: csrf.TemplateField(c.Request),
		})
	} else if c.Request.Method == "POST" {

		// Source
		file, err := c.FormFile("master_account_json")
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("get form err: %s", err.Error()))
			return
		}

		filename := filepath.Base(file.Filename)
		log.Printf("file uploaded %s", filename)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
			return
		}

		var newOrg dao.Organization
		if err := c.ShouldBind(&newOrg); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload binding: %s", err.Error()))
			return
		}
		newOrg.ID = daoHandler.GetNextUniqueId()

		if err := daoHandler.CreateOrganization(newOrg); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload creating db org: %s", err.Error()))
			return
		}

		c.HTML(http.StatusOK, "createOrg.tmpl", gin.H{
			"title": "createOrg - POST",
		})
	}
}

func LoginHandler(store sessions.Store, dao dao.DaoHandler, c *gin.Context) {
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
	session.Values["state"] = state
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	authenticator, err := auth.NewAuthenticator()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authenticator.Config.AuthCodeURL(state), http.StatusTemporaryRedirect)
}
