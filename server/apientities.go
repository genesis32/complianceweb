package server

type OrganizationTreeNode struct {
	// this needs to be a string because json sucks	and doesn't support 64 bit numbers
	ID       string                  `json:"id"`
	Name     string                  `json:"name"`
	Children []*OrganizationTreeNode `json:"children"`
}

type GcpServiceAccountCreateRequest struct {
	OwningOrganizationID       string
	CreatedByUserID            string
	OwningGcpProjectID         string
	RolesWithinOwningProjectID []string
}

type GcpServiceAccountCreateResponse struct {
	ID         string
	Credential string
}
