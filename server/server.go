package server

import (
	"fmt"
	"net/http"

	"github.com/genesis32/complianceweb/auth"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

func NewServer() *Server {
	sessionStore := sessions.NewFilesystemStore("", []byte("something-very-secret"))
	return &Server{SessionStore: sessionStore}
}

type Server struct {
	SessionStore sessions.Store
	router       *gin.Engine
}

func (s *Server) Serve() {
	router := gin.Default()
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/favicon.ico")

	router.LoadHTMLGlob("templates/html/**")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": "Welcome",
		})
	})
	router.GET("/login", func(c *gin.Context) {
		auth.LoginHandler(s.SessionStore, c.Writer, c.Request)
	})
	router.GET("/callback", func(c *gin.Context) {
		auth.CallbackHandler(s.SessionStore, c.Writer, c.Request)
	})
	router.GET("/user", func(c *gin.Context) {

		session, err := s.SessionStore.Get(c.Request, "auth-session")
		if err != nil {
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			return
		}

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": fmt.Sprintf("User: %+v", session.Values["profile"]),
		})
	})

	router.Run()
}
