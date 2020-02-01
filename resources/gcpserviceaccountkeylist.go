package resources

import "net/http"

type GcpServiceAccountKeyListResponse struct {
	Keys []string
}

type GcpServiceAccountResourceKeyListGetAction struct {
}

func (g GcpServiceAccountResourceKeyListGetAction) Name() string {
	return "Gcp Service Account List"
}

func (g GcpServiceAccountResourceKeyListGetAction) InternalKey() string {
	return "gcp.serviceaccount.keys"
}

func (g GcpServiceAccountResourceKeyListGetAction) Method() string {
	return "GET"
}

func (g GcpServiceAccountResourceKeyListGetAction) PermissionName() string {
	return "gcp.serviceaccount.read.execute"
}

func (g GcpServiceAccountResourceKeyListGetAction) Execute(w http.ResponseWriter, r *http.Request, params OperationParameters) *OperationResult {
	result := newOperationResult()

	mapAppParameters(params)

	return result
}
