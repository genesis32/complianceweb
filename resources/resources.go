package resources

import (
	"log"
	"net/http"

	"github.com/genesis32/complianceweb/dao"
)

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
	Path() string
	RequiredMetadata() []string
}

func NewOperationResult() *OperationResult {
	return &OperationResult{AuditMetadata: make(map[string]interface{}), AuditHumanReadable: "<<not defined>>"}
}

func FindResourceActions(internalKey string, loadedResources []OrganizationResourceAction) []OrganizationResourceAction {
	var ret []OrganizationResourceAction
	for _, v := range loadedResources {
		if internalKey == v.InternalKey() {
			ret = append(ret, v)
		}
	}
	return ret
}

func MapAppParameters(params OperationParameters) (dao.ResourceDaoHandler, []byte, int64) {
	daoHandler, ok := params["resourceDao"].(dao.ResourceDaoHandler)
	if !ok {
		log.Fatal("params['resourceDao'] not a ResourceDao type")
	}

	metadata, ok := params["organizationMetadata"].([]byte)
	if !ok {
		log.Fatal("params['organizationMetadata'] not a ResourceDao type")
	}
	organizationID, ok := params["organizationID"].(int64)
	if !ok {
		log.Fatal("params['organizationID'] not an int64 type")
	}
	return daoHandler, metadata, organizationID
}
