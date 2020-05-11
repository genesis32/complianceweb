package server

// BootstrapRequest contains initial information to make the app ready for use.
type BootstrapRequest struct {
	SystemAdminName string
}

// BootstrapResponse contains the response of the BootstrapRequest.
type BootstrapResponse struct {
	InviteCode int64 `json:",string,omitempty"`
	Href       string
}

type UserOrganizationResponse struct {
	ID       int64 `json:",string,omitempty"`
	Name     string
	Children []*UserOrganizationResponse
}

type OrganizationCreateRequest struct {
	ParentOrganizationID int64 `json:",string,omitempty"`
	Name                 string
}

type OrganizationCreateResponse struct {
	ID int64 `json:",string,omitempty"`
}

type GetOrganizationUserResponse struct {
	ID          int64 `json:",string,omitempty"`
	DisplayName string
	Roles       []UserOrgRoles
	Active      bool
}

// AddUserToOrganizationRequest adds a user to an organization.
// TODO(ddmassey): Do we maybe need to break them out by type?
type AddUserToOrganizationRequest struct {
	Name                 string
	ParentOrganizationID int64 `json:",string,omitempty"`
	RoleNames            []string
	CreateCredential     bool
}

// AddUserToOrganizationResponse is the response the server returns.
type AddUserToOrganizationResponse struct {
	InviteCode  int64 `json:",string,omitempty"`
	UserID      int64 `json:",string,omitempty"`
	Href        string
	Credentials string `json:",,omitempty"`
}

// UserOrgRoles provides the role names a user has an organization.
type UserOrgRoles struct {
	OrganizationID int64 `json:",string,omitempty"`
	RoleNames      []string
}

type UserUpdateRequest struct {
	Active bool
}

type SetRolesForUserRequest struct {
	Roles []UserOrgRoles
}

type RolesForUserResponse struct {
	Roles []UserOrgRoles
}

type OrganizationMetadataUpdateRequest struct {
	Metadata map[string]interface{}
}

type OrganizationMetadataResponse struct {
	Metadata map[string]interface{}
}
