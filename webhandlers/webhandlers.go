package webhandlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"

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

	stateWithInvite := strings.Split(r.URL.Query().Get("state"), "|")
	if len(stateWithInvite) > 1 {
		dao.InitUserFromInviteCode(stateWithInvite[1], fmt.Sprintf("%v", profile["sub"]))
	}

	// Redirect to logged in page
	http.Redirect(w, r, "/webapp/profile", http.StatusSeeOther)
}

func BootstrapHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.Redirect(302, fmt.Sprintf("/webapp/organization"))
}

func OrganizationModifyHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.SetCookie("X-CSRF-Token", csrf.Token(c.Request), 1000*60*5, "", "", false, false)
	orgId := c.Param("orgid")
	if c.Request.Method == "GET" {
		c.HTML(http.StatusOK, "modifyOrg.tmpl", gin.H{
			"title": fmt.Sprintf("Modify Org %s", orgId),
			"orgId": orgId,
		})
	}
}

func InviteHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	if c.Request.Method == "GET" {
		inviteCode := c.Param("inviteCode")
		theUser, err := daoHandler.LoadUserFromInviteCode(inviteCode)
		if err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("error getting invite code: %s", err.Error()))
			return
		}
		if theUser == nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("invite code not valid"))
			return
		}
		c.Redirect(302, fmt.Sprintf("/webapp/login?inviteCode=%s", inviteCode))
	}
}

func UsersJsonHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	if c.Request.Method == "GET" {
		r := make(map[string]interface{})
		r["foo"] = "GET"
		c.JSON(200, r)
	} else if c.Request.Method == "POST" {

		var frm AddUserToOrganizationForm

		if err := c.ShouldBind(&frm); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload format: %s", err.Error()))
			return
		}

		inviteCode, _ := daoHandler.CreateInviteForUser(frm.OrganizationId, frm.Name)

		r := make(map[string]string)
		r["inviteCode"] = inviteCode
		c.JSON(200, r)
	}
}

func OrganizationCreateHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	if c.Request.Method == "GET" {
		c.HTML(http.StatusOK, "createOrg.tmpl", gin.H{
			"title":          "Create Org",
			csrf.TemplateTag: csrf.TemplateField(c.Request),
		})
	} else if c.Request.Method == "POST" {

		var orgForm OrganizationForm
		if err := c.ShouldBind(&orgForm); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload binding: %s", err.Error()))
			return
		}
		var newOrg dao.Organization
		newOrg.ID = daoHandler.GetNextUniqueId()
		newOrg.DisplayName = orgForm.Name
		newOrg.MasterAccountType = "GCP"
		newOrg.EncodeMasterAccountCredential(orgForm.RetrieveContents())

		if err := daoHandler.CreateOrganization(&newOrg); err != nil {
			c.String(http.StatusBadRequest, fmt.Sprintf("upload creating db org: %s", err.Error()))
			return
		}
		c.Redirect(302, fmt.Sprintf("/webapp/organization/%d", newOrg.ID))
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

	authenticator, err := auth.NewAuthenticator()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authenticator.Config.AuthCodeURL(state), http.StatusTemporaryRedirect)
}
