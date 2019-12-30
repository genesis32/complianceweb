package server

import (
	"encoding/gob"
	"log"
	"net/http"

	"github.com/genesis32/complianceweb/auth"
	"github.com/genesis32/complianceweb/dao"
	"github.com/genesis32/complianceweb/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

// Server contains all the server code
type Server struct {
	Config        *ServerConfiguration
	Dao           dao.DaoHandler
	SessionStore  sessions.Store
	Authenticator *auth.Authenticator
	router        *gin.Engine
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

func loadConfiguration(daoHandler dao.DaoHandler) *ServerConfiguration {
	ret := &ServerConfiguration{}

	dbSettings, err := daoHandler.GetSettings(CookieAuthenticationKeyConfigurationKey, CookieEncryptionKeyConfigurationKey)
	if err != nil {
		log.Fatalf("error getting settings: %v", err)
	}
	if len(dbSettings) == 0 {
		ret.CookieAuthenticationKey, ret.CookieEncryptionKey = initCookieKeys(daoHandler)
	} else {
		ret.CookieAuthenticationKey = dbSettings[CookieAuthenticationKeyConfigurationKey].Base64DecodeValue()
		ret.CookieEncryptionKey = dbSettings[CookieEncryptionKeyConfigurationKey].Base64DecodeValue()
	}

	return ret
}

// NewServer returns a new server
func NewServer() *Server {
	dao := dao.NewDaoHandler()
	dao.Open()
	dao.TrySelect()

	config := loadConfiguration(dao)

	// We aren't even using this anymore but we'll keep it around just incase
	sessionStore := sessions.NewCookieStore(config.CookieAuthenticationKey, config.CookieEncryptionKey)
	sessionStore.Options.MaxAge = 0

	authenticator, err := auth.NewAuthenticator()
	if err != nil {
		log.Fatal(err)
	}
	return &Server{Config: config, SessionStore: sessionStore, Dao: dao, Authenticator: authenticator}
}

// Startup the server
func (s *Server) Startup() error {
	gob.Register(map[string]interface{}{})
	gob.Register(&dao.OrganizationUser{})

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

// Serve the traffic
func (s *Server) Serve() {
	s.router = gin.Default()
	s.router.MaxMultipartMemory = 8 << 20 // 8 MiB
	s.router.Static("/static", "./static")
	s.router.StaticFile("/favicon.ico", "./static/favicon.ico")

	s.router.LoadHTMLGlob("templates/html/*.tmpl")

	webapp := s.router.Group("webapp")
	{
		webapp.GET("/", s.registerWebApp(IndexHandler))
		webapp.GET("/invite/:inviteCode", s.registerWebApp(InviteHandler))
		webapp.GET("/login", s.registerWebApp(LoginHandler))
		webapp.GET("/callback", s.registerWebApp(CallbackHandler))
	}

	apiRoutes := s.router.Group("/api")
	apiRoutes.Use(validOIDCTokenRequired(s))
	{
		apiRoutes.POST("/bootstrap", s.registerWebApp(BootstrapApiPostHandler))

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
