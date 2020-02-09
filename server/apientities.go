package server

type BootstrapRequest struct {
	SystemAdminName string
}

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
}

type AddUserToOrganizationRequest struct {
	Name                 string
	ParentOrganizationID int64 `json:",string,omitempty"`
	RoleNames            []string
}

type AddUserToOrganizationResponse struct {
	InviteCode int64 `json:",string,omitempty"`
	Href       string
}

type UserOrgRoles struct {
	OrganizationID int64 `json:",string,omitempty"`
	RoleNames      []string
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
