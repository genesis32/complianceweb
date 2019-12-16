package server

type UserOrganizationResponse struct {
	ID       int64                       `json:",string,omitempty"`
	Name     string                      `json:"name"`
	Children []*UserOrganizationResponse `json:"children"`
}

type GcpServiceAccountCreateRequest struct {
	OwningOrganizationID string
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
