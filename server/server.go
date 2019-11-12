package server

import (
	"encoding/gob"
	"net/http"

	"github.com/genesis32/complianceweb/dao"
	"github.com/genesis32/complianceweb/webhandlers"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	adapter "github.com/gwatts/gin-adapter"
)

// Server contains all the server code
type Server struct {
	Dao          dao.DaoHandler
	SessionStore sessions.Store
	router       *gin.Engine
}

// NewServer returns a new server
func NewServer() *Server {
	sessionStore := sessions.NewFilesystemStore("", []byte("something-very-secret"))
	dao := dao.NewDaoHandler()
	return &Server{SessionStore: sessionStore, Dao: dao}
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

type webAppFunc func(s sessions.Store, dao dao.DaoHandler, c *gin.Context)

func (s *Server) registerWebApp(fn webAppFunc) func(c *gin.Context) {
	return (func(c *gin.Context) {
		fn(s.SessionStore, s.Dao, c)
	})
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

func authenticationRequired(store sessions.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := store.Get(c.Request, "auth-session")
		if err != nil {
			c.String(http.StatusUnauthorized, "Not authorized")
			c.Abort()
		}
		_, ok := session.Values["organization_user"].(*dao.OrganizationUser)
		if !ok {
			c.String(http.StatusUnauthorized, "Not authorized")
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
		webapp.GET("/", s.registerWebApp(webhandlers.IndexHandler))
		webapp.GET("/login", s.registerWebApp(webhandlers.LoginHandler))
		webapp.GET("/callback", s.registerWebApp(webhandlers.CallbackHandler))
		webapp.GET("/profile", s.registerWebApp(webhandlers.ProfileHandler))

		webapp.GET("/bootstrap", s.registerWebApp(webhandlers.BootstrapHandler))

		webapp.GET("/organization", s.registerWebApp(webhandlers.OrganizationCreateHandler))
		webapp.POST("/organization", s.registerWebApp(webhandlers.OrganizationCreateHandler))

		webapp.GET("/organization/:orgid", s.registerWebApp(webhandlers.OrganizationModifyHandler))
		webapp.POST("/organization/:orgid", s.registerWebApp(webhandlers.OrganizationModifyHandler))

		webapp.GET("/invite/:inviteCode", s.registerWebApp(webhandlers.InviteHandler))

		webapp.GET("/userJSON", s.registerWebApp(webhandlers.UsersJsonHandler))
		webapp.POST("/userJSON", s.registerWebApp(webhandlers.UsersJsonHandler))
	}
	authenticatedRoutes := webapp.Group("/user")
	authenticatedRoutes.Use(authenticationRequired(s.SessionStore))
	{
		authenticatedRoutes.GET("/", s.registerWebApp(webhandlers.UserIndexHandler))
	}

	s.router.GET("/", func(c *gin.Context) {
		c.Redirect(301, "/webapp/")
	})

	s.router.Run()
}
