package webhandlers

import (
	"net/http"

	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

func UserIndexHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	session, _ := store.Get(c.Request, "auth-session")
	t := session.Values["organization_user"].(*dao.OrganizationUser)
	c.HTML(http.StatusOK, "userIndex.tmpl", gin.H{
		"userId": t.ID,
	})
}
