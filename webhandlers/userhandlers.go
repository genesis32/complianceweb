package webhandlers

import (
	"fmt"
	"net/http"

	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
)

type OrganizationTreeNode struct {
	ID       int64                   `json:"id"`
	Name     string                  `json:"name"`
	Children []*OrganizationTreeNode `json:"children"`
}

func UserOrganizationApiHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	if c.Request.Method == "GET" {
		c1 := &OrganizationTreeNode{ID: 1, Name: "child1", Children: []*OrganizationTreeNode{}}
		treeRoot := &OrganizationTreeNode{ID: 2, Name: "foobar", Children: []*OrganizationTreeNode{c1}}

		c.JSON(http.StatusOK, treeRoot)
	} else if c.Request.Method == "POST" {

	}
}

func UserIndexHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.SetCookie("X-CSRF-Token", csrf.Token(c.Request), 1000*60*5, "", "", false, false)

	session, _ := store.Get(c.Request, "auth-session")
	t := session.Values["organization_user"].(*dao.OrganizationUser)

	c.HTML(http.StatusOK, "userIndex.tmpl", gin.H{
		"dataz": fmt.Sprintf("OrgUser:%+v", t),
	})
}
