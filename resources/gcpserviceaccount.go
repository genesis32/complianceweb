package resources

import "log"

type GcpServiceAccountResourcePostAction struct {
}

func (g GcpServiceAccountResourcePostAction) Method() string {
	return "POST"
}

func (g GcpServiceAccountResourcePostAction) Allowed(permissions []string) bool {
	panic("implement me")
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

type GcpServiceAccountResourceGetAction struct {
}

func (g GcpServiceAccountResourceGetAction) Name() string {
	return "GCP Service Account Manager Retrieve"
}

func (g GcpServiceAccountResourceGetAction) InternalKey() string {
	return "gcp.serviceaccount"
}

func (g GcpServiceAccountResourceGetAction) Method() string {
	return "GET"
}

func (g GcpServiceAccountResourceGetAction) Allowed(permissions []string) bool {
	panic("implement me")
}

func (g GcpServiceAccountResourceGetAction) PermissionName() string {
	return "gcp.serviceaccount.read.execute"
}

func (g GcpServiceAccountResourceGetAction) Execute(params OperationParameters) {
	log.Printf("gcp.serviceaccount get")
}
