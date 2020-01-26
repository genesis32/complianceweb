package resources

import (
	"log"
	"net/http"

	"github.com/genesis32/complianceweb/dao"
)

type GcpServiceAccountResourcePostAction struct{}

func (g GcpServiceAccountResourcePostAction) Method() string {
	return "POST"
}

func (g GcpServiceAccountResourcePostAction) PermissionName() string {
	return "gcp.serviceaccount.write.execute"
}

/*
	params["organizationID"] = organizationID
	params["organizationMetadata"] = metadata
	params["resourceDao"] = s.ResourceDao
	params["httpRequest"] = c.Request
	params["userUnfo"] = userInfo
*/
func (g GcpServiceAccountResourcePostAction) Execute(w http.ResponseWriter, r *http.Request, params OperationParameters) *OperationResult {
	result := newOperationResult()
	_, ok := params["resourceDao"].(dao.ResourceDaoHandler)
	if !ok {
		log.Fatal("params['resourceDao'] not a ResourceDao type")
	}
	log.Printf("gcp.serviceaccount post")

	w.WriteHeader(200)
	result.AuditHumanReadable = "This went ok"

	return result
}

func (g GcpServiceAccountResourcePostAction) Name() string {
	return "GCP Service Account Manager Create"
}

func (g GcpServiceAccountResourcePostAction) InternalKey() string {
	return "gcp.serviceaccount"
}
