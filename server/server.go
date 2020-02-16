package server

import (
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/genesis32/complianceweb/resources"

	"github.com/genesis32/complianceweb/auth"
	"github.com/genesis32/complianceweb/dao"
	"github.com/genesis32/complianceweb/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

// Server contains all the server code
type Server struct {
	Config              *ServerConfiguration
	Dao                 dao.DaoHandler
	ResourceDao         dao.ResourceDaoHandler
	SessionStore        sessions.Store
	Authenticator       auth.Authenticator
	router              *gin.Engine
	registeredResources dao.RegisteredResourcesStore
}

type WebappOperationMetadata map[string]interface{}
type WebAppOperationResult struct {
	AuditMetadata      WebappOperationMetadata
	AuditHumanReadable string
}

func initCookieKeys(daoHandler dao.DaoHandler) ([]byte, []byte) {
	authKey := utils.GenerateRandomBytes(32)
	encKey := utils.GenerateRandomBytes(32)
	authKeySetting := &dao.Setting{Key: CookieAuthenticationKeyConfigurationKey}
	authKeySetting.Base64EncodeValue(authKey)
	encKeySetting := &dao.Setting{Key: CookieEncryptionKeyConfigurationKey}
	encKeySetting.Base64EncodeValue(encKey)
	err := daoHandler.UpdateSettings(authKeySetting, encKeySetting)
	if err != nil {
		log.Fatal(err)
	}

	return authKey, encKey
}

func init() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
}

func loadConfiguration(daoHandler dao.DaoHandler) *ServerConfiguration {
	ret := &ServerConfiguration{}

	{
		dbSettings := daoHandler.GetSettings(CookieAuthenticationKeyConfigurationKey, CookieEncryptionKeyConfigurationKey)
		if len(dbSettings) == 0 {
			ret.CookieAuthenticationKey, ret.CookieEncryptionKey = initCookieKeys(daoHandler)
		} else {
			ret.CookieAuthenticationKey = dbSettings[CookieAuthenticationKeyConfigurationKey].Base64DecodeValue()
			ret.CookieEncryptionKey = dbSettings[CookieEncryptionKeyConfigurationKey].Base64DecodeValue()
		}
	}

	{
		dbSettings := daoHandler.GetSettings(OIDCIssuerBaseUrlConfigurationKey, Auth0ClientIdConfigurationKey, Auth0ClientSecretConfigurationKey, SystemBaseUrlConfigurationKey)
		if len(dbSettings) != 4 {
			log.Fatal("parameters not loaded. Do all oidc configuration parameters exist in the db?")
		}
		ret.OIDCIssuer = dbSettings[OIDCIssuerBaseUrlConfigurationKey].Value
		ret.Auth0ClientID = dbSettings[Auth0ClientIdConfigurationKey].Value
		ret.Auth0ClientSecret = dbSettings[Auth0ClientSecretConfigurationKey].Value
		ret.SystemBaseUrl = dbSettings[SystemBaseUrlConfigurationKey].Value
	}

	return ret
}

// NewServer returns a new server
func NewServer() *Server {
	rand.Seed(time.Now().UnixNano())

	daoHandler := dao.NewDaoHandler(nil)
	daoHandler.Open()
	daoHandler.TrySelect()

	resourceDaoHandler := dao.NewResourceDaoHandler(nil)
	resourceDaoHandler.Open()

	config := loadConfiguration(daoHandler)

	// We aren't even using this anymore but we'll keep it around just incase
	sessionStore := sessions.NewCookieStore(config.CookieAuthenticationKey, config.CookieEncryptionKey)
	sessionStore.Options.MaxAge = 0

	callbackUrl := fmt.Sprintf("%s/webapp/callback", config.SystemBaseUrl)

	var authenticator auth.Authenticator
	if v, ok := os.LookupEnv("ENV"); ok && v == "test" {
		authenticator = auth.NewTestAuthenticator()
	} else {
		authenticator = auth.NewAuth0Authenticator(callbackUrl, config.OIDCIssuer, config.Auth0ClientID, config.Auth0ClientSecret)
	}

	return &Server{Config: config, SessionStore: sessionStore, Dao: daoHandler, ResourceDao: resourceDaoHandler, Authenticator: authenticator}
}

// Shutdown the server
func (s *Server) Shutdown() error {
	err := s.Dao.Close()
	return err
}

type webAppFunc func(t *dao.OrganizationUser, s *Server, store sessions.Store, dao dao.DaoHandler, c *gin.Context) *WebAppOperationResult
type resourceApiFunc func(w http.ResponseWriter, r *http.Request, parameters resources.OperationParameters) *resources.OperationResult

func (s *Server) registerWebApp(fn webAppFunc) func(c *gin.Context) {
	return s.registerWebAppA(true, fn)
}

func (s *Server) registerWebAppA(authenticationRequired bool, fn webAppFunc) func(c *gin.Context) {
	return func(c *gin.Context) {
		var userInfo *dao.OrganizationUser
		if authenticationRequired {
			subject, ok := c.Get("authenticated_user_profile")
			if !ok {
				c.String(http.StatusForbidden, "User does not exist")
				return
			}

			userInfo = s.Dao.LoadUserFromCredential(subject.(utils.OpenIDClaims)["sub"].(string), dao.UserActiveState)
			if userInfo == nil {
				c.String(http.StatusForbidden, "User does not exist")
				return
			}
		}

		auditRecord := dao.NewAuditRecord("webapp", c.Request.Method)
		auditRecord.OrganizationUserID = 0
		if userInfo != nil {
			auditRecord.OrganizationUserID = userInfo.ID
		}
		auditRecord.OrganizationID = 0 // TODO: Fix this

		s.Dao.CreateAuditRecord(auditRecord)

		operationResult := fn(userInfo, s, s.SessionStore, s.Dao, c)

		// TODO: Fix this so it's required in the future
		if operationResult != nil {
			auditRecord.Metadata = newWebappAuditMetadata(operationResult.AuditMetadata)
			auditRecord.HumanReadable = operationResult.AuditHumanReadable
		}

		s.Dao.SealAuditRecord(auditRecord)
	}
}

func (s *Server) registerResourceApi(resourceAction resources.OrganizationResourceAction, fn resourceApiFunc) func(c *gin.Context) {
	return func(c *gin.Context) {
		organizationID, err := utils.StringToInt64(c.Param("organizationID"))
		if err != nil {
			c.String(http.StatusBadRequest, "expected numeric organization identifier")
			return
		}

		subject, exists := c.Get("authenticated_user_profile")
		if !exists {
			c.String(http.StatusForbidden, "")
			return
		}

		userInfo := s.Dao.LoadUserFromCredential(subject.(utils.OpenIDClaims)["sub"].(string), dao.UserActiveState)
		if userInfo == nil {
			c.String(http.StatusForbidden, "User does not exist")
			return
		}

		hasPermission := s.Dao.DoesUserHavePermission(userInfo.ID, organizationID, resourceAction.PermissionName())

		// @gmail.com as orgs
		//		log.Printf("resource organizationid: %d required permission: %s user: %d", organizationID, resourceAction.PermissionName(), userInfo.ID)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
			return
		}

		var metadataBytes []byte

		/* TODO: For now just pull the first one from the list (next return a metadata that is the intersction of everything in this list */
		requiredMetadata := resourceAction.RequiredMetadata()
		if len(requiredMetadata) > 0 {
			_, metadataBytes = s.Dao.LoadMetadataInTree(organizationID, requiredMetadata[0])
			if len(metadataBytes) == 0 {
				c.String(http.StatusBadRequest, "no metadata present")
				return
			}
		}
		//		log.Printf("loaded orgid: %d metadata: %v", orgIDWithMetadata, metadata)

		params := resources.OperationParameters{}
		params["organizationID"] = organizationID
		params["organizationMetadata"] = metadataBytes
		params["resourceDao"] = s.ResourceDao
		params["userInfo"] = userInfo

		auditRecord := dao.NewAuditRecord(resourceAction.InternalKey(), resourceAction.Method())
		auditRecord.OrganizationUserID = userInfo.ID
		auditRecord.OrganizationID = organizationID

		s.Dao.CreateAuditRecord(auditRecord)

		operationResult := fn(c.Writer, c.Request, params)

		// TODO: Fix this so it's required in the future
		if operationResult != nil {
			auditRecord.Metadata = newResourceAuditMetadata(operationResult.AuditMetadata)
			auditRecord.HumanReadable = operationResult.AuditHumanReadable
		}

		s.Dao.SealAuditRecord(auditRecord)
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

func (s *Server) Initialize() *gin.Engine {

	gob.Register(map[string]interface{}{})
	gob.Register(&dao.OrganizationUser{})

	s.registeredResources = s.Dao.LoadEnabledResources()

	if k, exists := os.LookupEnv("ENV"); exists && k == "test" {
		s.router = gin.New()
	} else {
		s.router = gin.Default()
	}

	s.router.MaxMultipartMemory = 8 << 20 // 8 MiB
	s.router.Static("/static", "./static")
	s.router.StaticFile("/favicon.ico", "./static/favicon.ico")

	s.router.LoadHTMLGlob("templates/html/*.tmpl")

	s.router.GET("/", func(c *gin.Context) {
		c.Redirect(301, "/webapp/")
	})

	system := s.router.Group("/system")
	{
		system.POST("/bootstrap", s.registerWebAppA(false, BootstrapApiPostHandler))
	}

	webapp := s.router.Group("/webapp")
	{
		webapp.GET("/", s.registerWebApp(IndexHandler))
		webapp.GET("/invite/:inviteCode", s.registerWebApp(InviteHandler))
		webapp.GET("/login", s.registerWebApp(LoginHandler))
		webapp.GET("/callback", s.registerWebApp(CallbackHandler))
	}

	apiRoutes := s.router.Group("/api")
	apiRoutes.Use(validOIDCTokenRequired(s))
	{

		apiRoutes.POST("/organizations", s.registerWebApp(OrganizationApiPostHandler))
		apiRoutes.GET("/organizations", s.registerWebApp(OrganizationApiGetHandler))
		apiRoutes.GET("/organizations/:organizationID", s.registerWebApp(OrganizationDetailsApiGetHandler))

		apiRoutes.PUT("/organizations/:organizationID/metadata", s.registerWebApp(OrganizationMetadataApiPutHandler))
		apiRoutes.GET("/organizations/:organizationID/metadata", s.registerWebApp(OrganizationMetadataApiGetHandler))

		apiRoutes.POST("/users", s.registerWebApp(UserApiPostHandler))
		apiRoutes.GET("/users/:userID", s.registerWebApp(UserApiGetHandler))
		apiRoutes.DELETE("/users/:userID", s.registerWebApp(UserApiDeleteHandler))

		apiRoutes.PUT("/users/:userID/roles", s.registerWebApp(UserRoleApiPostHandler))
	}

	resourceRoutes := apiRoutes.Group("/resources/:organizationID")
	for _, r := range s.registeredResources {
		keyResources := resources.FindResourceActions(r.InternalKey, loadedResources)
		for _, theResource := range keyResources {
			path := theResource.Path()
			if len(strings.TrimSpace(path)) > 0 {
				if path[0] != '/' {
					path = fmt.Sprintf("%s/%s", theResource.InternalKey(), path)
				} else {
					path = fmt.Sprintf("%s%s", theResource.InternalKey(), path)
				}
			} else {
				path = fmt.Sprintf("%s", theResource.InternalKey())
			}
			resourceRoutes.Handle(theResource.Method(),
				path,
				s.registerResourceApi(theResource, theResource.Execute))
		}
	}

	return s.router
}

// Serve the traffic
func (s *Server) Serve() {
	err := s.router.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// TODO: Make it work on a glob?
func newResourceAuditMetadata(metadata resources.OperationMetadata) dao.AuditMetadata {
	ret := make(dao.AuditMetadata)
	for k, v := range metadata {
		ret[k] = v
	}
	return ret
}

// TODO: Make it work on a glob?
func newWebappAuditMetadata(metadata WebappOperationMetadata) dao.AuditMetadata {
	ret := make(dao.AuditMetadata)
	for k, v := range metadata {
		ret[k] = v
	}
	return ret
}
