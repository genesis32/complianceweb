package server

import (
	"fmt"
	"net/http"

	"github.com/genesis32/complianceweb/auth"
	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

type Server struct {
	Dao          dao.DaoHandler
	SessionStore sessions.Store
	router       *gin.Engine
}

func NewServer() *Server {
	sessionStore := sessions.NewFilesystemStore("", []byte("something-very-secret"))
	dao := dao.NewDaoHandler()
	return &Server{SessionStore: sessionStore, Dao: dao}
}

func (s *Server) Startup() error {
	dbOpenErr := s.Dao.Open()
	if dbOpenErr != nil {
		return dbOpenErr
	}
	s.Dao.TrySelect()
	return nil
}

func (s *Server) Serve() {
	s.router = gin.Default()
	s.router.Static("/static", "./static")
	s.router.StaticFile("/favicon.ico", "./static/favicon.ico")

	s.router.LoadHTMLGlob("templates/html/**")
	s.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": "Welcome",
		})
	})
	s.router.GET("/login", func(c *gin.Context) {
		auth.LoginHandler(s.SessionStore, c.Writer, c.Request)
	})
	s.router.GET("/callback", func(c *gin.Context) {
		auth.CallbackHandler(s.SessionStore, c.Writer, c.Request)
	})
	s.router.GET("/user", func(c *gin.Context) {

		session, err := s.SessionStore.Get(c.Request, "auth-session")
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			return
		}

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": fmt.Sprintf("User: %+v", session.Values["profile"]),
		})
	})

	s.router.Run()
}
