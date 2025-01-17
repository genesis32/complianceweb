package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/genesis32/complianceweb/utils"

	"github.com/genesis32/complianceweb/dao"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

func createInviteLink(baseUrl string, inviteCode int64, daoHandler dao.DaoHandler) string {
	var href string
	if baseUrl == "" {
		configKeys := daoHandler.GetSettings(SystemBaseURLConfigurationKey)
		href = fmt.Sprintf("%s/webapp/login?inviteCode=%v", configKeys[SystemBaseURLConfigurationKey].Value, inviteCode)
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

func BootstrapApiPostHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {

	configKeys := daoHandler.GetSettings(BootstrapConfigurationKey, SystemBaseURLConfigurationKey)
	if len(configKeys) == 0 || configKeys[BootstrapConfigurationKey].Value != "true" {
		c.String(http.StatusMethodNotAllowed, fmt.Sprintf("not allowed"))
		return nil
	}

	var bootstrapRequest BootstrapRequest
	if err := c.ShouldBind(&bootstrapRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("bootstrap binding: %s", err.Error()))
		return nil
	}

	var response BootstrapResponse
	userId, inviteCode := daoHandler.CreateInviteForUser(0, bootstrapRequest.SystemAdminName)

	daoHandler.SetRolesToUser(0, userId, []string{"System Admin"})

	response.InviteCode = inviteCode
	response.Href = createInviteLink(configKeys[SystemBaseURLConfigurationKey].Value, inviteCode, daoHandler)

	c.JSON(200, response)
	return nil
}

func OrganizationApiPostHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {

	var createRequest OrganizationCreateRequest
	if err := c.ShouldBind(&createRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("binding: %s", err.Error()))
		return nil
	}

	if createRequest.ParentOrganizationID != 0 {
		// Make sure that user has visibility over a ParentOrganizationID
		hasPermission := daoHandler.DoesUserHavePermission(t.ID, createRequest.ParentOrganizationID, OrganizationCreatePermission)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
			return nil
		}
	} else if createRequest.ParentOrganizationID == 0 {
		// Only a person with system permission is allowed to create a root of a new tree
		hasPermission := daoHandler.DoesUserHaveSystemPermission(t.ID, SystemOrganizationCreatePermission)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
			return nil
		}
	}

	var newOrg dao.Organization
	newOrg.ID = utils.GetNextUniqueId()
	newOrg.DisplayName = createRequest.Name

	// TODO: Transaction?
	daoHandler.CreateOrganization(&newOrg)

	if createRequest.ParentOrganizationID != 0 {
		daoHandler.AssignOrganizationToParent(createRequest.ParentOrganizationID, newOrg.ID)
	}

	createResponse := &OrganizationCreateResponse{}
	createResponse.ID = newOrg.ID
	c.JSON(201, createResponse)
	return nil
}

func OrganizationDetailsApiGetHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {

	organizationIdStr := c.Param("organizationID")
	organizationId, _ := utils.StringToInt64(organizationIdStr)

	canView := daoHandler.CanUserViewOrg(t.ID, organizationId)
	if !canView {
		c.String(http.StatusUnauthorized, "not authorized")
		return nil
	}

	var queryFlags uint
	if daoHandler.DoesUserHavePermission(t.ID, organizationId, UserReadPermission) {
		queryFlags |= dao.UserReadExecutePermissionFlag
	}

	organization := daoHandler.LoadOrganizationDetails(organizationId, queryFlags)

	// TODO: Put into a nice public api response
	c.JSON(http.StatusOK, organization)
	return nil
}

func OrganizationApiGetHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {

	organizations := daoHandler.LoadOrganizationsForUser(t.ID)
	if len(organizations) == 0 {
		c.String(http.StatusBadRequest, "no organizations")
		return nil
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
	return nil
}

// UserAPIPostHandler creates a new User in the system (could be an application)
func UserAPIPostHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {
	var addRequest AddUserToOrganizationRequest

	if err := c.ShouldBind(&addRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload format: %s", err.Error()))
		return nil
	}

	if len(addRequest.RoleNames) == 0 {
		c.String(http.StatusBadRequest, "at least one role required")
		return nil
	}

	if !daoHandler.HasValidRoles(addRequest.RoleNames) {
		c.String(http.StatusBadRequest, "needs to contain all valid roles")
		return nil
	}

	hasPermission := daoHandler.DoesUserHavePermission(t.ID, addRequest.ParentOrganizationID, UserCreatePermission)
	if !hasPermission {
		// Are they a sys-admin?
		hasPermission = daoHandler.DoesUserHaveSystemPermission(t.ID, SystemUserCreatePermission)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
			return nil
		}
	}

	userId, inviteCode := daoHandler.CreateInviteForUser(addRequest.ParentOrganizationID, addRequest.Name)

	daoHandler.SetRolesToUser(addRequest.ParentOrganizationID, userId, addRequest.RoleNames)

	href := createInviteLink("", inviteCode, daoHandler)
	r := &AddUserToOrganizationResponse{InviteCode: inviteCode, Href: href, UserID: userId}
	c.JSON(http.StatusCreated, r)
	return nil
}

func OrganizationMetadataApiPutHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, handler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {
	var metadataUpdateRequest OrganizationMetadataUpdateRequest

	if err := c.ShouldBind(&metadataUpdateRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("metadata format: %s", err.Error()))
		return nil
	}

	organizationIDStr := c.Param("organizationID")
	organizationID, _ := utils.StringToInt64(organizationIDStr)

	hasPermission := handler.DoesUserHavePermission(t.ID, organizationID, OrganizationCreatePermission)
	if !hasPermission {
		c.String(http.StatusUnauthorized, "not authorized")
		return nil
	}

	handler.UpdateOrganizationMetadata(organizationID, metadataUpdateRequest.Metadata)
	return nil
}

func OrganizationMetadataApiGetHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, handler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {
	var metadataUpdateRequest OrganizationMetadataUpdateRequest

	if err := c.ShouldBind(&metadataUpdateRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("metadata format: %s", err.Error()))
		return nil
	}

	organizationIDStr := c.Param("organizationID")
	organizationID, _ := utils.StringToInt64(organizationIDStr)

	// TODO: Should we bound this by a permission?
	hasPermission := handler.CanUserViewOrg(t.ID, organizationID)
	if !hasPermission {
		c.String(http.StatusUnauthorized, "not authorized")
		return nil
	}

	metadata := handler.LoadOrganizationMetadata(organizationID)

	response := &OrganizationMetadataResponse{Metadata: metadata}

	// TODO: Make this into another object
	c.JSON(200, response)

	auditRecord := &WebAppOperationResult{}
	auditRecord.AuditHumanReadable = fmt.Sprintf("read metadata for organization: %d", organizationID)

	return auditRecord
}

func UserRoleApiPostHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, handler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {
	var rolesUpdateRequest SetRolesForUserRequest

	if err := c.ShouldBind(&rolesUpdateRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("roles update format: %s", err.Error()))
		return nil
	}

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
			return nil
		}
	}

	for _, r := range rolesUpdateRequest.Roles {
		handler.SetRolesToUser(r.OrganizationID, userID, r.RoleNames)
	}
	return nil
}

func MeApiGetHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, handler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {
	organizationUser := handler.LoadUserFromID(t.ID)
	response := GetOrganizationUserResponse{ID: organizationUser.ID, DisplayName: organizationUser.DisplayName, Active: organizationUser.CurrentState == dao.UserActiveState}
	for orgID, roles := range organizationUser.UserRoles {
		var roleNames []string
		for _, r := range roles {
			roleNames = append(roleNames, r.DisplayName)
		}
		response.Roles = append(response.Roles, UserOrgRoles{OrganizationID: orgID, RoleNames: roleNames})
	}
	c.JSON(http.StatusOK, response)
	return nil
}

func UserApiPutHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, handler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {
	var userUpdateRequest UserUpdateRequest

	if err := c.ShouldBind(&userUpdateRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("roles update format: %s", err.Error()))
		return nil
	}

	userIDStr := c.Param("userID")
	userID, err := utils.StringToInt64(userIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "user invalid ID")
		return nil
	}

	// TODO: We should return the same code regardless of whether you can't find the user
	// or you are not authorized to view.
	organizationUser := handler.LoadUserFromID(userID)
	if organizationUser == nil {
		c.String(http.StatusNotFound, "user not found")
		return nil
	}

	// user is not associated with any org (could be a sysadmin)
	if len(organizationUser.Organizations) == 0 {
		c.String(http.StatusUnauthorized, "not authorized")
		return nil
	}

	// if you don't have visibility over just one of the org don't allow this.
	for _, oid := range organizationUser.Organizations {
		userCanView := handler.CanUserViewOrg(userID, oid)
		if !userCanView {
			c.String(http.StatusUnauthorized, "not authorized")
			return nil
		}
		// Make sure the caller has permission to assign the role to this user.
		hasPermission := handler.DoesUserHavePermission(t.ID, oid, UserUpdatePermission)
		if !hasPermission {
			c.String(http.StatusUnauthorized, "not authorized")
			return nil
		}
	}

	switch {
	case userUpdateRequest.Active && (dao.UserDeactiveState == organizationUser.CurrentState):
		if organizationUser.ID == t.ID {
			c.String(http.StatusBadRequest, "not allowed to activate yourself")
			return nil
		}
		handler.UpdateUserState(organizationUser.ID, dao.UserActiveState)
		organizationUser.CurrentState = dao.UserActiveState
	case (userUpdateRequest.Active == false) && (dao.UserActiveState == organizationUser.CurrentState):
		if organizationUser.ID == t.ID {
			c.String(http.StatusBadRequest, "not allowed to deactivate yourself")
			return nil
		}
		handler.UpdateUserState(organizationUser.ID, dao.UserDeactiveState)
		organizationUser.CurrentState = dao.UserDeactiveState
	}

	c.Status(http.StatusOK)
	return nil
}

func UserApiGetHandler(t *dao.OrganizationUser, s *Server, store sessions.Store, handler dao.DaoHandler, c *gin.Context) *WebAppOperationResult {

	userIDStr := c.Param("userID")
	userID, _ := utils.StringToInt64(userIDStr)

	organizationUser := handler.LoadUserFromID(userID)
	response := GetOrganizationUserResponse{ID: organizationUser.ID, DisplayName: organizationUser.DisplayName, Active: organizationUser.CurrentState == dao.UserActiveState}
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
	return nil
}

/*
func UserCreateGcpServiceAccountApiPostHandler(s *Server, store sessions.Store, daoHandler dao.DaoHandler, c *gin.Context) {

	var serviceAccountRequest GcpServiceAccountCreateRequest
	if err := c.ShouldBind(&serviceAccountRequest); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("bad request: %s", err.Error()))
		return
	}

	subject, _ := c.Get("authenticated_user_profile")
	t, _ := daoHandler.LoadUserFromCredential(subject.(utils.OpenIDClaims)["sub"].(string))

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
