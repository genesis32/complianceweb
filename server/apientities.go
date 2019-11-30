package server

type UserOrganizationResponse struct {
	// this needs to be a string because json sucks	and doesn't support 64 bit numbers
	ID       string                      `json:"id"`
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
