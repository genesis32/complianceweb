package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/genesis32/complianceweb/auth"
	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

func contains(n *UserOrganizationResponse, children []*UserOrganizationResponse) bool {
	for _, ch := range children {
		if ch == n {
			return true
		}
	}
	return false
}

func BootstrapApiPostHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {

	var bootstrapRequest BootstrapRequest
	if err := c.ShouldBind(&bootstrapRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("bootstrap binding: %s", err.Error()))
		return
	}

	var response BootstrapResponse
	c.JSON(200, response)
}

func OrganizationApiPostHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {

	var createRequest OrganizationCreateRequest
	if err := c.ShouldBind(&createRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("binding: %s", err.Error()))
		return
	}

	subject, _ := c.Get("authenticated_user_profile")
	t, _ := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	// TODO: Add in test that user has visibility over a ParentOrganizationID
	if createRequest.ParentOrganizationID != 0 {
		hasPermission, _ := daoHandler.DoesUserHavePermission(t.ID, createRequest.ParentOrganizationID, OrganizationCreatePermission)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
			return
		}
	} else if createRequest.ParentOrganizationID == 0 {
		hasPermission, _ := daoHandler.DoesUserHaveSystemPermission(t.ID, SystemOrganizationCreatePermission)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
			return
		}
	}

	var newOrg dao.Organization
	newOrg.ID = daoHandler.GetNextUniqueId()
	newOrg.DisplayName = createRequest.Name
	newOrg.MasterAccountType = createRequest.AccountCredentialType
	newOrg.EncodeMasterAccountCredential(createRequest.AccountCredential)

	if err := daoHandler.CreateOrganization(&newOrg); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload creating db org: %s", err.Error()))
		return
	}

	if createRequest.ParentOrganizationID != 0 {
		daoHandler.AssignOrganizationToParent(createRequest.ParentOrganizationID, newOrg.ID)
	}

	createResponse := &OrganizationCreateResponse{}
	createResponse.ID = newOrg.ID
	c.JSON(201, createResponse)
}

func OrganizationDetailsApiGetHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	subject, _ := c.Get("authenticated_user_profile")
	t, _ := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	organizationIdStr := c.Param("organizationID")
	organizationId, _ := strconv.ParseInt(organizationIdStr, 10, 64)

	canView, _ := daoHandler.CanUserViewOrg(t.ID, organizationId)
	if !canView {
		c.String(http.StatusUnauthorized, "not authorized")
		return
	}

	organization, _ := daoHandler.LoadOrganizationDetails(organizationId)

	// TODO: Put into a nice public version
	c.JSON(http.StatusOK, organization)
}

func OrganizationApiGetHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	subject, _ := c.Get("authenticated_user_profile")

	t, _ := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	organizations, _ := daoHandler.LoadOrganizationsForUser(t.ID)

	orgTreeRep := make(map[int64]*UserOrganizationResponse)
	// all the organizations we can see
	for k, v := range organizations {
		orgTreeRep[k] = &UserOrganizationResponse{Name: v.DisplayName, ID: k, Children: []*UserOrganizationResponse{}}
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
}

func UserApiPostHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	var addRequest AddUserToOrganizationRequest

	if err := c.ShouldBind(&addRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload format: %s", err.Error()))
		return
	}

	inviteCode, _ := daoHandler.CreateInviteForUser(addRequest.ParentOrganizationID, addRequest.Name)
	// TODO: Handle error

	href := fmt.Sprintf("http://localhost:3000/webapp/login?inviteCode=%v", inviteCode)
	r := &AddUserToOrganizationResponse{InviteCode: inviteCode, Href: href}
	c.JSON(200, r)
}

func UserCreateGcpServiceAccountApiPostHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {

	var serviceAccountRequest GcpServiceAccountCreateRequest
	if err := c.ShouldBind(&serviceAccountRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("bad request: %s", err.Error()))
		return
	}

	subject, _ := c.Get("authenticated_user_profile")
	t, _ := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	canView, _ := daoHandler.CanUserViewOrg(t.ID, serviceAccountRequest.OwningOrganizationID)

	if !canView {
		c.String(http.StatusUnauthorized, "not authorized")
		return
	}

	response := &GcpServiceAccountCreateResponse{}

	serviceAccountCredentials, err := daoHandler.LoadServiceAccountCredentials(serviceAccountRequest.OwningOrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	serviceAccountKey, _ := createServiceAccount(context.Background(), serviceAccountCredentials.RawCredentials, serviceAccountRequest.DisplayName)

	if serviceAccountKey != nil {
		response.ID = serviceAccountKey.Name
	}
	c.JSON(http.StatusOK, response)
}
