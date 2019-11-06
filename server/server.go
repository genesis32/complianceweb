package server

import (
	"github.com/genesis32/complianceweb/dao"
	"github.com/genesis32/complianceweb/webhandlers"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
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

// Serve the traffic
func (s *Server) Serve() {
	s.router = gin.Default()
	s.router.Static("/static", "./static")
	s.router.StaticFile("/favicon.ico", "./static/favicon.ico")

	s.router.LoadHTMLGlob("templates/html/**")

	webapp := s.router.Group("webapp")
	{
		webapp.GET("/", s.registerWebApp(webhandlers.IndexHandler))
		webapp.GET("/login", s.registerWebApp(webhandlers.LoginHandler))
		webapp.GET("/callback", s.registerWebApp(webhandlers.CallbackHandler))
		webapp.GET("/profile", s.registerWebApp(webhandlers.ProfileHandler))
	}

	s.router.GET("/", func(c *gin.Context) {
		c.Redirect(301, "/webapp/")
	})

	s.router.Run()
}
