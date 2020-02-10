package gcp

import "google.golang.org/api/iam/v1"

type ServiceAccountCreateRequest struct {
	UniqueIdentifier string
	ProjectID        string
	Roles            []string
}

type ServiceAccountCreateResponse struct {
	UniqueIdentifier string
}

type ServiceAccountKeyCreateRequest struct {
	GcpEmailIdentifier string
}

type ServiceAccountKeyCreateResponse struct {
	UniqueIdentifier string
}

type ServiceAccountKeyGetResponse struct {
	Key *iam.ServiceAccountKey
}
