package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/genesis32/complianceweb/utils"

	"github.com/genesis32/complianceweb/auth"
	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

func createInviteLink(baseUrl string, inviteCode int64, daoHandler dao.DaoHandler) string {
	var href string
	if baseUrl == "" {
		configKeys := daoHandler.GetSettings(SystemBaseUrlConfigurationKey)
		href = fmt.Sprintf("%s/webapp/login?inviteCode=%v", configKeys[SystemBaseUrlConfigurationKey].Value, inviteCode)
		return href
	} else {
		href = fmt.Sprintf("%s/webapp/login?inviteCode=%v", baseUrl, inviteCode)
	}
	return href
}

func contains(n *UserOrganizationResponse, children []*UserOrganizationResponse) bool {
	for _, ch := range children {
		if ch == n {
			return true
		}
	}
	return false
}

func BootstrapApiPostHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {

	configKeys := daoHandler.GetSettings(BootstrapConfigurationKey, SystemBaseUrlConfigurationKey)
	if len(configKeys) == 0 || configKeys[BootstrapConfigurationKey].Value != "true" {
		c.String(http.StatusMethodNotAllowed, fmt.Sprintf("not allowed"))
		return
	}

	var bootstrapRequest BootstrapRequest
	if err := c.ShouldBind(&bootstrapRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("bootstrap binding: %s", err.Error()))
		return
	}

	var response BootstrapResponse
	userId, inviteCode := daoHandler.CreateInviteForUser(0, bootstrapRequest.SystemAdminName)

	daoHandler.SetRolesToUser(0, userId, []string{"System Admin"})

	response.InviteCode = inviteCode
	response.Href = createInviteLink(configKeys[SystemBaseUrlConfigurationKey].Value, inviteCode, daoHandler)

	c.JSON(200, response)
}

func OrganizationApiPostHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {

	var createRequest OrganizationCreateRequest
	if err := c.ShouldBind(&createRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("binding: %s", err.Error()))
		return
	}

	subject, _ := c.Get("authenticated_user_profile")
	t := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	if createRequest.ParentOrganizationID != 0 {
		// Make sure that user has visibility over a ParentOrganizationID
		hasPermission := daoHandler.DoesUserHavePermission(t.ID, createRequest.ParentOrganizationID, OrganizationCreatePermission)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
			return
		}
	} else if createRequest.ParentOrganizationID == 0 {
		// Only a person with system permission is allowed to create a root of a new tree
		hasPermission := daoHandler.DoesUserHaveSystemPermission(t.ID, SystemOrganizationCreatePermission)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
			return
		}
	}

	var newOrg dao.Organization
	newOrg.ID = daoHandler.GetNextUniqueId()
	newOrg.DisplayName = createRequest.Name

	// TODO: Transaction?
	daoHandler.CreateOrganization(&newOrg)

	if createRequest.ParentOrganizationID != 0 {
		daoHandler.AssignOrganizationToParent(createRequest.ParentOrganizationID, newOrg.ID)
	}

	createResponse := &OrganizationCreateResponse{}
	createResponse.ID = newOrg.ID
	c.JSON(201, createResponse)
}

func OrganizationDetailsApiGetHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	subject, _ := c.Get("authenticated_user_profile")
	t := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	organizationIdStr := c.Param("organizationID")
	organizationId, _ := utils.StringToInt64(organizationIdStr)

	canView := daoHandler.CanUserViewOrg(t.ID, organizationId)
	if !canView {
		c.String(http.StatusUnauthorized, "not authorized")
		return
	}

	var queryFlags uint
	if daoHandler.DoesUserHavePermission(t.ID, organizationId, UserReadPermission) {
		queryFlags |= dao.UserReadExecutePermissionFlag
	}

	organization := daoHandler.LoadOrganizationDetails(organizationId, queryFlags)

	// TODO: Put into a nice public api response
	c.JSON(http.StatusOK, organization)
}

func OrganizationApiGetHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {
	subject, _ := c.Get("authenticated_user_profile")

	t := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	organizations := daoHandler.LoadOrganizationsForUser(t.ID)
	if len(organizations) == 0 {
		c.String(http.StatusBadRequest, "no organizations")
		return
	}

	orgTreeRep := make(map[int64]*UserOrganizationResponse)
	// all the organizations we can see
	for k, v := range organizations {
		orgTreeRep[k] = &UserOrganizationResponse{Name: v.DisplayName, ID: k, Children: []*UserOrganizationResponse{}}
	}

	for k := range orgTreeRep {
		pathPieces := strings.Split(organizations[k].Path, ".")
		for i := range pathPieces {
			if i > 0 {
				parentID, _ := utils.StringToInt64(pathPieces[i-1])
				// if we can't see the parent just disregard even mapping it..
				if orgTreeRep[parentID] == nil {
					continue
				}
				pathID, _ := utils.StringToInt64(pathPieces[i])
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

	subject, _ := c.Get("authenticated_user_profile")

	t := daoHandler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	if len(addRequest.Roles) == 0 {
		c.String(http.StatusBadRequest, "at least one role required")
		return
	}

	if !daoHandler.HasValidRoles(addRequest.Roles) {
		c.String(http.StatusBadRequest, "contains at least one invalid role")
		return
	}

	hasPermission := daoHandler.DoesUserHavePermission(t.ID, addRequest.ParentOrganizationID, UserCreatePermission)
	if !hasPermission {
		c.String(http.StatusUnauthorized, "not authorized")
		return
	}

	userId, inviteCode := daoHandler.CreateInviteForUser(addRequest.ParentOrganizationID, addRequest.Name)

	daoHandler.SetRolesToUser(addRequest.ParentOrganizationID, userId, addRequest.Roles)

	href := createInviteLink("", inviteCode, daoHandler)
	r := &AddUserToOrganizationResponse{InviteCode: inviteCode, Href: href}
	c.JSON(200, r)
}

func OrganizationMetadataApiPutHandler(s *Server, store sessions.Store, handler dao.DaoHandler, c *gin.Context) {
	var metadataUpdateRequest OrganizationMetadataUpdateRequest

	if err := c.ShouldBind(&metadataUpdateRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("metadata format: %s", err.Error()))
		return
	}

	subject, _ := c.Get("authenticated_user_profile")
	t := handler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	organizationIDStr := c.Param("organizationID")
	organizationID, _ := utils.StringToInt64(organizationIDStr)

	hasPermission := handler.DoesUserHavePermission(t.ID, organizationID, OrganizationCreatePermission)
	if !hasPermission {
		c.String(http.StatusUnauthorized, "not authorized")
		return
	}

	handler.UpdateOrganizationMetadata(organizationID, metadataUpdateRequest.Metadata)
}

func OrganizationMetadataApiGetHandler(s *Server, store sessions.Store, handler dao.DaoHandler, c *gin.Context) {
	var metadataUpdateRequest OrganizationMetadataUpdateRequest

	if err := c.ShouldBind(&metadataUpdateRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("metadata format: %s", err.Error()))
		return
	}

	subject, _ := c.Get("authenticated_user_profile")
	t := handler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	organizationIDStr := c.Param("organizationID")
	organizationID, _ := utils.StringToInt64(organizationIDStr)

	// TODO: Break out permissions
	hasPermission := handler.DoesUserHavePermission(t.ID, organizationID, OrganizationCreatePermission)
	if !hasPermission {
		c.String(http.StatusUnauthorized, "not authorized")
		return
	}

	settings := handler.LoadOrganizationMetadata(organizationID)
	c.JSON(200, settings)
}

func UserRoleApiPostHandler(s *Server, store sessions.Store, handler dao.DaoHandler, c *gin.Context) {
	var rolesUpdateRequest SetRolesForUserRequest

	if err := c.ShouldBind(&rolesUpdateRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("roles update format: %s", err.Error()))
		return
	}

	subject, _ := c.Get("authenticated_user_profile")
	t := handler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	userIDStr := c.Param("userID")
	userID, _ := utils.StringToInt64(userIDStr)

	for _, r := range rolesUpdateRequest.Roles {
		// Make sure the userID has visibility to this org
		userCanView := handler.CanUserViewOrg(userID, r.OrganizationID)
		if !userCanView {
			c.String(http.StatusUnauthorized, "not authorized")
		}
		// Make sure the caller has permission to assign the role to this user.
		hasPermission := handler.DoesUserHavePermission(t.ID, r.OrganizationID, UserUpdatePermission)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
		}
		// Make sure all roles passed in are valid
		if !handler.HasValidRoles(r.RoleNames) {
			c.String(http.StatusBadRequest, "contains at least one invalid role.")
			return
		}
	}

	for _, r := range rolesUpdateRequest.Roles {
		handler.SetRolesToUser(r.OrganizationID, userID, r.RoleNames)
	}
}

func UserApiGetHandler(s *Server, store sessions.Store, handler dao.DaoHandler, c *gin.Context) {
	subject, _ := c.Get("authenticated_user_profile")
	t := handler.LoadUserFromCredential(subject.(auth.OpenIDClaims)["sub"].(string))

	userIDStr := c.Param("userID")
	userID, _ := utils.StringToInt64(userIDStr)

	organizationUser := handler.LoadUserFromID(userID)
	response := GetOrganizationUserResponse{ID: organizationUser.ID, DisplayName: organizationUser.DisplayName}
	for orgID, roles := range organizationUser.UserRoles {
		// don't return roles belonging to orgs the user isn't part of
		if !handler.CanUserViewOrg(t.ID, orgID) {
			continue
		}
		var roleNames []string
		for _, r := range roles {
			roleNames = append(roleNames, r.DisplayName)
		}
		response.Roles = append(response.Roles, UserOrgRoles{OrganizationID: orgID, RoleNames: roleNames})
	}
	c.JSON(http.StatusOK, response)
}

/*
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

	serviceAccountKey, _ := resources.createServiceAccount(context.Background(), serviceAccountCredentials.RawCredentials, serviceAccountRequest.DisplayName)

	if serviceAccountKey != nil {
		response.ID = serviceAccountKey.Name
	}
	c.JSON(http.StatusOK, response)
}
*/
