package resources

import "net/http"

type GcpServiceAccountResourceListGetAction struct {
}

func (g GcpServiceAccountResourceListGetAction) Name() string {
	return "Gcp Service Account List"
}

func (g GcpServiceAccountResourceListGetAction) InternalKey() string {
	return "gcp.serviceaccount"
}

func (g GcpServiceAccountResourceListGetAction) Method() string {
	return "GET"
}

func (g GcpServiceAccountResourceListGetAction) PermissionName() string {
	return "gcp.serviceaccount.write.execute"
}

func (g GcpServiceAccountResourceListGetAction) Execute(w http.ResponseWriter, r *http.Request, params OperationParameters) *OperationResult {
	result := newOperationResult()

	mapAppParameters(params)

	return result
}
