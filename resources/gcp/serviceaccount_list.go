package gcp

import (
	"net/http"

	"github.com/genesis32/complianceweb/resources"
)

type ServiceAccountResourceListGetAction struct {
}

func (g ServiceAccountResourceListGetAction) Path() string {
	return ""
}

func (g ServiceAccountResourceListGetAction) Name() string {
	return "Gcp Service Account List"
}

func (g ServiceAccountResourceListGetAction) InternalKey() string {
	return "gcp.serviceaccount"
}

func (g ServiceAccountResourceListGetAction) Method() string {
	return "GET"
}

func (g ServiceAccountResourceListGetAction) PermissionName() string {
	return "gcp.serviceaccount.read.execute"
}

func (g ServiceAccountResourceListGetAction) Execute(w http.ResponseWriter, r *http.Request, params resources.OperationParameters) *resources.OperationResult {
	result := resources.NewOperationResult()

	mapAppParameters(params)

	return result
}
