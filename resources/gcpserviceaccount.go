package resources

import "log"

type GcpServiceAccountResourcePostAction struct {
}

func (g GcpServiceAccountResourcePostAction) Method() string {
	return "POST"
}

func (g GcpServiceAccountResourcePostAction) PermissionName() string {
	return "gcp.serviceaccount.write.execute"
}

func (g GcpServiceAccountResourcePostAction) Execute(params OperationParameters) {
	log.Printf("gcp.serviceaccount post")
}

func (g GcpServiceAccountResourcePostAction) Name() string {
	return "GCP Service Account Manager Create"
}

func (g GcpServiceAccountResourcePostAction) InternalKey() string {
	return "gcp.serviceaccount"
}
