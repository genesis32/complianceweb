package server

import (
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
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

type OperationMetadata map[string]interface{}
type OperationResult struct {
	AuditMetadata      OperationMetadata
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
		log.Fatalf("%v", err)
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
			panic("parameters not loaded. Do all oidc configuration parameters exist in the db?")
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
	if v, ok := os.LookupEnv("ENV"); ok && v == "dev" {
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

type webAppFunc func(s *Server, store sessions.Store, dao dao.DaoHandler, c *gin.Context) *OperationResult
type resourceApiFunc func(w http.ResponseWriter, r *http.Request, parameters resources.OperationParameters) *resources.OperationResult

func (s *Server) registerWebApp(fn webAppFunc) func(c *gin.Context) {
	return func(c *gin.Context) {
		var userId int64
		subject, ok := c.Get("authenticated_user_profile")
		if ok {
			// TODO: Move the userinfo parameter to the method it's calling
			userInfo := s.Dao.LoadUserFromCredential(subject.(utils.OpenIDClaims)["sub"].(string))
			userId = userInfo.ID
		}

		auditRecord := dao.NewAuditRecord("webapp", c.Request.Method)
		auditRecord.OrganizationUserID = userId
		auditRecord.OrganizationID = 0 // TODO: Fix this

		s.Dao.CreateAuditRecord(auditRecord)

		operationResult := fn(s, s.SessionStore, s.Dao, c)

		// TODO: Fix this so it's required in the future
		if operationResult != nil {
			auditRecord.Metadata = operationResult.AuditMetadata
			auditRecord.HumanReadable = operationResult.AuditHumanReadable
		}

		s.Dao.SealAuditRecord(auditRecord)
	}
}

func (s *Server) registerResourceApi(resourceAction resources.OrganizationResourceAction, fn resourceApiFunc) func(c *gin.Context) {
	return func(c *gin.Context) {
		subject, _ := c.Get("authenticated_user_profile")
		userInfo := s.Dao.LoadUserFromCredential(subject.(utils.OpenIDClaims)["sub"].(string))

		organizationID, _ := utils.StringToInt64(c.Param("organizationID"))
		hasPermission := s.Dao.DoesUserHavePermission(userInfo.ID, organizationID, resourceAction.PermissionName())

		// @gmail.com as orgs
		//		log.Printf("resource organizationid: %d required permission: %s user: %d", organizationID, resourceAction.PermissionName(), userInfo.ID)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
			return
		}

		_, metadata := s.Dao.LoadMetadataInTree(organizationID, "gcpCredentials")
		//		log.Printf("loaded orgid: %d metadata: %v", orgIDWithMetadata, metadata)

		params := resources.OperationParameters{}
		params["organizationID"] = organizationID
		params["organizationMetadata"] = metadata
		params["resourceDao"] = s.ResourceDao
		params["userUnfo"] = userInfo

		auditRecord := dao.NewAuditRecord(resourceAction.InternalKey(), resourceAction.Method())
		auditRecord.OrganizationUserID = userInfo.ID
		auditRecord.OrganizationID = organizationID

		s.Dao.CreateAuditRecord(auditRecord)

		operationResult := fn(c.Writer, c.Request, params)

		auditRecord.Metadata = operationResult.AuditMetadata
		auditRecord.HumanReadable = operationResult.AuditHumanReadable

		s.Dao.SealAuditRecord(auditRecord)
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

func (s *Server) Initialize() *gin.Engine {

	gob.Register(map[string]interface{}{})
	gob.Register(&dao.OrganizationUser{})

	s.registeredResources = s.Dao.LoadEnabledResources()

	s.router = gin.Default()
	s.router.MaxMultipartMemory = 8 << 20 // 8 MiB
	s.router.Static("/static", "./static")
	s.router.StaticFile("/favicon.ico", "./static/favicon.ico")

	s.router.LoadHTMLGlob("templates/html/*.tmpl")

	system := s.router.Group("/system")
	{
		system.POST("/bootstrap", s.registerWebApp(BootstrapApiPostHandler))
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

		apiRoutes.PUT("/users/:userID/roles", s.registerWebApp(UserRoleApiPostHandler))
	}

	resourceRoutes := apiRoutes.Group("/resources/:organizationID")
	for _, r := range s.registeredResources {
		resources := resources.FindResourceActions(r.InternalKey)
		for _, theResource := range resources {
			resourceRoutes.Handle(theResource.Method(),
				theResource.InternalKey(),
				s.registerResourceApi(theResource, theResource.Execute))
		}
	}

	s.router.GET("/", func(c *gin.Context) {
		c.Redirect(301, "/webapp/")
	})

	return s.router
}

// Serve the traffic
func (s *Server) Serve() {
	s.router.Run()
}
