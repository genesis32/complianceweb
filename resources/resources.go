package resources

import (
	"net/http"
)

var loadedResources = []OrganizationResourceAction{
	&GcpServiceAccountResourcePostAction{},
	&GcpServiceAccountResourceKeyPostAction{},
	&GcpServiceAccountResourceListGetAction{},
}

type OperationParameters map[string]interface{}
type OperationMetadata map[string]interface{}

type OperationResult struct {
	AuditMetadata      OperationMetadata
	AuditHumanReadable string
}

type OrganizationResourceAction interface {
	Name() string
	InternalKey() string
	Method() string
	PermissionName() string
	Execute(w http.ResponseWriter, r *http.Request, params OperationParameters) *OperationResult
}

func newOperationResult() *OperationResult {
	return &OperationResult{AuditMetadata: make(map[string]interface{}), AuditHumanReadable: "<<not defined>>"}
}

func FindResourceActions(internalKey string) []OrganizationResourceAction {
	var ret []OrganizationResourceAction
	for _, v := range loadedResources {
		if internalKey == v.InternalKey() {
			ret = append(ret, v)
		}
	}
	return ret
}
