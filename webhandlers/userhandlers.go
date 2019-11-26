package webhandlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
)

func UserIndexHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.SetCookie("X-CSRF-Token", csrf.Token(c.Request), 1000*60*5, "", "", false, false)

	session, _ := store.Get(c.Request, "auth-session")
	t := session.Values["organization_user"].(*dao.OrganizationUser)

	c.HTML(http.StatusOK, "userIndex.tmpl", gin.H{
		"dataz": fmt.Sprintf("OrgUser:%+v", t),
	})
}

func UserOrganizationViewHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.SetCookie("X-CSRF-Token", csrf.Token(c.Request), 1000*60*5, "", "", false, false)

	organizationIdStr := c.Param("organizationId")
	organizationId, _ := strconv.ParseInt(organizationIdStr, 10, 64)

	session, _ := store.Get(c.Request, "auth-session")
	theUser := session.Values["organization_user"].(*dao.OrganizationUser)

	theOrganization, _ := daoHandler.LoadOrganization(theUser.ID, organizationId)

	c.HTML(http.StatusOK, "userOrganization.tmpl", gin.H{
		"organizationName": fmt.Sprintf("%s", theOrganization.DisplayName),
		"orgId":            organizationIdStr,
	})
}
