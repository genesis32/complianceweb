package webhandlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
)

type OrganizationTreeNode struct {
	// this needs to be a string because json sucks	and doesn't support 64 bit numbers
	ID       string                  `json:"id"`
	Name     string                  `json:"name"`
	Children []*OrganizationTreeNode `json:"children"`
}

func contains(n *OrganizationTreeNode, children []*OrganizationTreeNode) bool {
	for _, ch := range children {
		if ch == n {
			return true
		}
	}
	return false
}

func UserOrganizationApiHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	if c.Request.Method == "GET" {

		session, _ := store.Get(c.Request, "auth-session")
		t := session.Values["organization_user"].(*dao.OrganizationUser)
		organizations, _ := daoHandler.LoadOrganizationsForUser(t.ID)

		orgTreeRep := make(map[int64]*OrganizationTreeNode)
		// all the organizations we can see
		for k, v := range organizations {
			jsonFormatInt64 := strconv.FormatInt(k, 10)
			orgTreeRep[k] = &OrganizationTreeNode{Name: v.DisplayName, ID: jsonFormatInt64, Children: []*OrganizationTreeNode{}}
		}

		for k := range orgTreeRep {
			pathPieces := strings.Split(organizations[k].Path, ".")
			for i := range pathPieces {
				if i > 0 {
					parentID, _ := strconv.ParseInt(pathPieces[i-1], 10, 64)
					// if we can't see the parent just disregard even mapping it..
					if orgTreeRep[parentID] == nil {
						continue
					}
					pathID, _ := strconv.ParseInt(pathPieces[i], 10, 64)
					if !contains(orgTreeRep[pathID], orgTreeRep[parentID].Children) {
						orgTreeRep[parentID].Children = append(orgTreeRep[parentID].Children, orgTreeRep[pathID])
					}
				}
			}
		}
		// hack for now.. single node and just return where in the tree it's visible from
		treeRoot := orgTreeRep[t.Organizations[0]]

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

func UserOrganizationViewHandler(store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	c.SetCookie("X-CSRF-Token", csrf.Token(c.Request), 1000*60*5, "", "", false, false)

	organizationIdStr := c.Param("organizationId")
	organizationId, _ := strconv.ParseInt(organizationIdStr, 10, 64)

	session, _ := store.Get(c.Request, "auth-session")
	theUser := session.Values["organization_user"].(*dao.OrganizationUser)

	theOrganization, _ := daoHandler.LoadOrganization(theUser.ID, organizationId)

	c.HTML(http.StatusOK, "userOrganization.tmpl", gin.H{
		"organizationName": fmt.Sprintf("%s", theOrganization.DisplayName),
	})
}
