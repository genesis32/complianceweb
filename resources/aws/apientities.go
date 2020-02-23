package aws

import "github.com/aws/aws-sdk-go/service/iam"

type IAMUserCreateRequest struct {
	UserName string `validate:"min=4,max=16,regexp=^[A-Za-z0-9]*$"`
}

type IAMUserCreateResponse struct {
	ID       int64 `json:",string,omitempty"`
	UserName string
}

type IAMUserApproveRequest struct {
	ID       int64 `json:",string,omitempty"`
	UserName string
}

type IAMUserApproveResponse struct {
	ID               int64 `json:",string,omitempty"`
	Approved         bool
	CreateUserOutput *iam.CreateUserOutput `json:",,omitempty"`
}
