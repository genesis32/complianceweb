package gcp

import (
	"net/http"

	"github.com/genesis32/complianceweb/resources"
)

type GcpServiceAccountResourceListGetAction struct {
}

func (g GcpServiceAccountResourceListGetAction) Path() string {
	return ""
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
	return "gcp.serviceaccount.read.execute"
}

func (g GcpServiceAccountResourceListGetAction) Execute(w http.ResponseWriter, r *http.Request, params resources.OperationParameters) *resources.OperationResult {
	result := resources.NewOperationResult()

	mapAppParameters(params)

	return result
}
