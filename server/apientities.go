package server

type UserOrganizationResponse struct {
	ID       int64 `json:",string,omitempty"`
	Name     string
	Children []*UserOrganizationResponse
}

type GcpServiceAccountCreateRequest struct {
	OwningOrganizationID int64 `json:",string,omitempty"`
	OwningGcpProjectID   string
	DisplayName          string
}

type GcpServiceAccountCreateResponse struct {
	ID         string
	State      string
	Credential string
}

type OrganizationCreateRequest struct {
	ParentOrganizationID  int64 `json:",string,omitempty"`
	Name                  string
	AccountCredentialType string
	AccountCredential     string
}

type OrganizationCreateResponse struct {
	ID int64 `json:",string,omitempty"`
}

type AddUserToOrganizationRequest struct {
	Name                 string
	ParentOrganizationID int64 `json:",string,omitempty"`
}

type AddUserToOrganizationResponse struct {
	InviteCode string
	Href       string
}

type AssignRoleToUserRequest struct {
}
