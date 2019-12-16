package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/genesis32/complianceweb/auth"
	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	adapter "github.com/gwatts/gin-adapter"
)

// Server contains all the server code
type Server struct {
	Dao           dao.DaoHandler
	SessionStore  sessions.Store
	Authenticator *auth.Authenticator
	router        *gin.Engine
}

// NewServer returns a new server
func NewServer() *Server {
	sessionStore := sessions.NewFilesystemStore("", []byte("something-very-secret"))
	dao := dao.NewDaoHandler()
	authenticator, err := auth.NewAuthenticator()
	if err != nil {
		panic(err)
	}
	return &Server{SessionStore: sessionStore, Dao: dao, Authenticator: authenticator}
}

// Startup the server
func (s *Server) Startup() error {
	gob.Register(map[string]interface{}{})
	gob.Register(&dao.OrganizationUser{})

	dbOpenErr := s.Dao.Open()
	if dbOpenErr != nil {
		return dbOpenErr
	}
	s.Dao.TrySelect()
	return nil
}

// Shutdown the server
func (s *Server) Shutdown() error {
	err := s.Dao.Close()
	return err
}

type webAppFunc func(s *Server, store sessions.Store, dao dao.DaoHandler, c *gin.Context)

func (s *Server) registerWebApp(fn webAppFunc) func(c *gin.Context) {
	return func(c *gin.Context) {
		fn(s, s.SessionStore, s.Dao, c)
	}
}

func Logger() gin.HandlerFunc {

	return func(c *gin.Context) {

		statusCode := c.Writer.Status()
		if statusCode >= 400 {
			//ok this is an request with error, let's make a record for it
			//log body here
		}
	}
}

func validOIDCTokenRequired(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader("Authorization")

		if authorizationHeader != "" {
			profile, err := s.Authenticator.ValidateAuthorizationHeader(authorizationHeader)
			if err == nil && profile != nil {
				c.Set("authenticated_user_profile", profile)
				c.Next()
				return
			}
		}
		c.String(http.StatusUnauthorized, "Not authorized")
		c.Abort()
	}
}

func authenticationRequired(store sessions.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := store.Get(c.Request, "auth-session")
		if err != nil {
			c.String(http.StatusUnauthorized, "Not authenticated")
			c.Abort()
		}
		_, ok := session.Values["organization_user"].(*dao.OrganizationUser)
		if !ok {
			c.String(http.StatusUnauthorized, "Not authenticated")
			c.Abort()
		}
		c.Next()
	}
}

// Serve the traffic
func (s *Server) Serve() {
	s.router = gin.Default()
	s.router.MaxMultipartMemory = 8 << 20 // 8 MiB
	s.router.Static("/static", "./static")
	s.router.StaticFile("/favicon.ico", "./static/favicon.ico")

	s.router.LoadHTMLGlob("templates/html/*.tmpl")

	csrfMiddleware := csrf.Protect([]byte("32-byte-long-auth-key"), csrf.Secure(false), csrf.HttpOnly(false), csrf.Path("/"))

	webapp := s.router.Group("webapp")
	webapp.Use(adapter.Wrap(csrfMiddleware))
	{
		webapp.GET("/", s.registerWebApp(IndexHandler))
		webapp.GET("/invite/:inviteCode", s.registerWebApp(InviteHandler))
		webapp.GET("/login", s.registerWebApp(LoginHandler))
		webapp.GET("/callback", s.registerWebApp(CallbackHandler))
		webapp.GET("/bootstrap", s.registerWebApp(BootstrapHandler))
	}
	authenticatedRoutes := webapp.Group("/user")
	authenticatedRoutes.Use(authenticationRequired(s.SessionStore))
	{
		authenticatedRoutes.GET("/", s.registerWebApp(UserIndexHandler))
		authenticatedRoutes.GET("/organization/:organizationId", s.registerWebApp(UserOrganizationViewHandler))
	}

	apiRoutes := s.router.Group("/api")
	apiRoutes.Use(validOIDCTokenRequired(s))
	{
		apiRoutes.POST("/organizations", s.registerWebApp(OrganizationApiPostHandler))
		apiRoutes.GET("/organizations", s.registerWebApp(OrganizationApiGetHandler))
		apiRoutes.GET("/organizations/:organizationID", s.registerWebApp(OrganizationDetailsApiGetHandler))

		apiRoutes.POST("/users", s.registerWebApp(UserApiPostHandler))

		apiRoutes.POST("/gcp/service-account", s.registerWebApp(UserCreateGcpServiceAccountApiPostHandler))
	}

	s.router.GET("/", func(c *gin.Context) {
		c.Redirect(301, "/webapp/")
	})

	s.router.Run()
}

func IndexHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {

	c.HTML(http.StatusOK, "index.tmpl", gin.H{
		"title": "Welcome",
	})
}

func ProfileHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	session, err := store.Get(c.Request, "auth-session")
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	t := session.Values["organization_user"].(*dao.OrganizationUser)
	c.HTML(http.StatusOK, "profile.tmpl", gin.H{
		"title": fmt.Sprintf("User: %+v userid:%d", session.Values["profile"], t.ID),
	})
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

func BootstrapHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.Redirect(302, fmt.Sprintf("/webapp/organization"))
}

func OrganizationModifyHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.SetCookie("X-CSRF-Token", csrf.Token(c.Request), 1000*60*5, "", "", false, false)
	orgId := c.Param("orgid")
	if c.Request.Method == "GET" {
		c.HTML(http.StatusOK, "modifyOrg.tmpl", gin.H{
			"title": fmt.Sprintf("Modify Org %s", orgId),
			"orgId": orgId,
		})
	}
}

func InviteHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
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

func UsersJsonHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
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

func OrganizationCreateHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
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

func contains(n *UserOrganizationResponse, children []*UserOrganizationResponse) bool {
	for _, ch := range children {
		if ch == n {
			return true
		}
	}
	return false
}

func OrganizationApiPostHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {

	var createRequest OrganizationCreateRequest
	if err := c.ShouldBind(&createRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload binding: %s", err.Error()))
		return
	}
	var newOrg dao.Organization
	newOrg.ID = daoHandler.GetNextUniqueId()
	newOrg.DisplayName = createRequest.Name
	newOrg.MasterAccountType = createRequest.AccountCredentialType
	newOrg.EncodeMasterAccountCredential(createRequest.AccountCredential)

	if err := daoHandler.CreateOrganization(&newOrg); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload creating db org: %s", err.Error()))
		return
	}

	if createRequest.ParentOrganizationID != 0 {
		daoHandler.AssignOrganizationToParent(createRequest.ParentOrganizationID, newOrg.ID)
	}

	createResponse := &OrganizationCreateResponse{}
	createResponse.ID = newOrg.ID
	c.JSON(201, createResponse)
}

func OrganizationDetailsApiGetHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	//subject, _ := c.Get("authenticated_user_profile")
	//	t, _ := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	organizationIdStr := c.Param("organizationID")
	organizationId, _ := strconv.ParseInt(organizationIdStr, 10, 64)

	organization, _ := daoHandler.LoadOrganization(organizationId)

	c.JSON(http.StatusOK, organization)
}

func OrganizationApiGetHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	subject, _ := c.Get("authenticated_user_profile")

	t, _ := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	organizations, _ := daoHandler.LoadOrganizationsForUser(t.ID)

	orgTreeRep := make(map[int64]*UserOrganizationResponse)
	// all the organizations we can see
	for k, v := range organizations {
		orgTreeRep[k] = &UserOrganizationResponse{Name: v.DisplayName, ID: k, Children: []*UserOrganizationResponse{}}
	}

	for k := range orgTreeRep {
		pathPieces := strings.Split(organizations[k].Path, ".")
		for i := range pathPieces {
			if i > 0 {
				parentID, _ := strconv.ParseInt(pathPieces[i-1], 10, 64)
				// if we can't see the parent just disregard even mapping it..
				if orgTreeRep[parentID] == nil {
					continue
				}
				pathID, _ := strconv.ParseInt(pathPieces[i], 10, 64)
				if !contains(orgTreeRep[pathID], orgTreeRep[parentID].Children) {
					orgTreeRep[parentID].Children = append(orgTreeRep[parentID].Children, orgTreeRep[pathID])
				}
			}
		}
	}
	// hack for now.. single node and just return where in the tree it's visible from
	treeRoot := orgTreeRep[t.Organizations[0]]

	c.JSON(http.StatusOK, treeRoot)
}

func UserApiPostHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	var addRequest AddUserToOrganizationRequest

	if err := c.ShouldBind(&addRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload format: %s", err.Error()))
		return
	}

	inviteCode, _ := daoHandler.CreateInviteForUser(addRequest.ParentOrganizationID, addRequest.Name)

	r := make(map[string]string)
	r["inviteCode"] = inviteCode
	r["href"] = fmt.Sprintf("http://localhost:3000/webapp/login?inviteCode=%v", inviteCode)
	c.JSON(200, r)
}

func UserCreateGcpServiceAccountApiPostHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {

	var req GcpServiceAccountCreateRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("bad request: %s", err.Error()))
		return
	}
	owningOrganizationID, _ := strconv.ParseInt(req.OwningOrganizationID, 10, 64)

	subject, _ := c.Get("authenticated_user_profile")
	t, _ := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	canView, _ := daoHandler.CanUserViewOrg(t.ID, owningOrganizationID)

	if !canView {
		c.String(http.StatusUnauthorized, "not authorized")
		return
	}

	response := &GcpServiceAccountCreateResponse{}

	serviceAccountCredentials, err := daoHandler.LoadServiceAccountCredentials(owningOrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	serviceAccountKey, _ := createServiceAccount(context.Background(), serviceAccountCredentials.RawCredentials, req.DisplayName)

	if serviceAccountKey != nil {
		response.ID = serviceAccountKey.Name
	}
	c.JSON(http.StatusOK, response)
}

func UserIndexHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.SetCookie("X-CSRF-Token", csrf.Token(c.Request), 1000*60*5, "", "", false, false)

	session, _ := store.Get(c.Request, "auth-session")
	t := session.Values["organization_user"].(*dao.OrganizationUser)

	c.HTML(http.StatusOK, "userIndex.tmpl", gin.H{
		"dataz": fmt.Sprintf("OrgUser:%+v", t),
	})
}

func UserOrganizationViewHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.SetCookie("X-CSRF-Token", csrf.Token(c.Request), 1000*60*5, "", "", false, false)

	organizationIdStr := c.Param("organizationId")
	organizationId, _ := strconv.ParseInt(organizationIdStr, 10, 64)

	theOrganization, _ := daoHandler.LoadOrganization(organizationId)

	c.HTML(http.StatusOK, "userOrganization.tmpl", gin.H{
		"organizationName": fmt.Sprintf("%s", theOrganization.DisplayName),
		"orgId":            organizationIdStr,
	})
}
