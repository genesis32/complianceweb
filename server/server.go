package server

import (
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/genesis32/complianceweb/auth"
	"github.com/genesis32/complianceweb/dao"
	"github.com/genesis32/complianceweb/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

// Server contains all the server code
type Server struct {
	Config              *Configuration
	Dao                 dao.DaoHandler
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

type webAppFunc func(t *dao.OrganizationUser, s *Server, store sessions.Store, dao dao.DaoHandler, c *gin.Context) *WebAppOperationResult

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

func loadConfiguration(daoHandler dao.DaoHandler) *Configuration {
	ret := &Configuration{}

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
		dbSettings := daoHandler.GetSettings(OIDCIssuerBaseURLConfigurationKey, Auth0ClientIDConfigurationKey, Auth0ClientSecretConfigurationKey, SystemBaseURLConfigurationKey)
		if len(dbSettings) != 4 {
			log.Fatal("parameters not loaded. Do all oidc configuration parameters exist in the db?")
		}
		ret.OIDCIssuer = dbSettings[OIDCIssuerBaseURLConfigurationKey].Value
		ret.Auth0ClientID = dbSettings[Auth0ClientIDConfigurationKey].Value
		ret.Auth0ClientSecret = dbSettings[Auth0ClientSecretConfigurationKey].Value
		ret.SystemBaseUrl = dbSettings[SystemBaseURLConfigurationKey].Value
	}

	return ret
}

// NewServer returns a new server
func NewServer() *Server {
	rand.Seed(time.Now().UnixNano())

	daoHandler := dao.NewDaoHandler(nil)
	daoHandler.Open()
	daoHandler.TrySelect()

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

	return &Server{Config: config, SessionStore: sessionStore, Dao: daoHandler, Authenticator: authenticator}
}

// Shutdown the server
func (s *Server) Shutdown() error {
	err := s.Dao.Close()
	return err
}

func (s *Server) registerAPI(fn webAppFunc) func(c *gin.Context) {
	return s.registerAPIA(true, fn)
}

func (s *Server) registerAPIA(authenticationRequired bool, fn webAppFunc) func(c *gin.Context) {
	return func(c *gin.Context) {
		var userInfo *dao.OrganizationUser
		if authenticationRequired {
			subject, ok := c.Get("authenticated_user_profile")
			if !ok {
				c.String(http.StatusForbidden, "User credential not supplied.")
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
		system.POST("/bootstrap", s.registerAPIA(false, BootstrapApiPostHandler))
	}

	webapp := s.router.Group("/webapp")
	{
		webapp.GET("/", s.registerAPIA(false, IndexHandler))
		webapp.GET("/invite/:inviteCode", s.registerAPIA(false, InviteHandler))
		webapp.GET("/login", s.registerAPIA(false, LoginHandler))
		webapp.GET("/callback", s.registerAPIA(false, CallbackHandler))
	}

	apiRoutes := s.router.Group("/api")
	apiRoutes.Use(validOIDCTokenRequired(s))
	{

		apiRoutes.POST("/organizations", s.registerAPI(OrganizationApiPostHandler))
		apiRoutes.GET("/organizations", s.registerAPI(OrganizationApiGetHandler))
		apiRoutes.GET("/organizations/:organizationID", s.registerAPI(OrganizationDetailsApiGetHandler))

		apiRoutes.PUT("/organizations/:organizationID/metadata", s.registerAPI(OrganizationMetadataApiPutHandler))
		apiRoutes.GET("/organizations/:organizationID/metadata", s.registerAPI(OrganizationMetadataApiGetHandler))

		apiRoutes.POST("/users", s.registerAPI(UserAPIPostHandler))
		apiRoutes.GET("/users/:userID", s.registerAPI(UserApiGetHandler))
		apiRoutes.GET("/me", s.registerAPI(MeApiGetHandler))
		apiRoutes.PUT("/users/:userID", s.registerAPI(UserApiPutHandler))
		apiRoutes.PUT("/users/:userID/roles", s.registerAPI(UserRoleApiPostHandler))
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
func newWebappAuditMetadata(metadata WebappOperationMetadata) dao.AuditMetadata {
	ret := make(dao.AuditMetadata)
	for k, v := range metadata {
		ret[k] = v
	}
	return ret
}
