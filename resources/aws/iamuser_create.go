package aws

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/genesis32/complianceweb/resources"
)

type resourceMetadata struct {
	AWSCredentials awsCredentials
}
type awsCredentials struct {
	AccessKeyID     string
	AccessKeySecret string
}

type IAMUserCreateResourcePostAction struct {
}

func (I IAMUserCreateResourcePostAction) RequiredMetadata() []string {
	return []string{"awsCredentials"}
}

func (I IAMUserCreateResourcePostAction) Name() string {
	return "AWS IAM User Create"
}

func (I IAMUserCreateResourcePostAction) InternalKey() string {
	return "aws.iam.user"
}

func (I IAMUserCreateResourcePostAction) Method() string {
	return "POST"
}

func (I IAMUserCreateResourcePostAction) PermissionName() string {
	return "aws.iam.user.create.execute"
}

func (I IAMUserCreateResourcePostAction) Execute(w http.ResponseWriter, r *http.Request, params resources.OperationParameters) *resources.OperationResult {
	result := resources.NewOperationResult()

	_, metadata, _ := resources.MapAppParameters(params)

	var resourceMetadata resourceMetadata
	if err := json.Unmarshal(metadata, &resourceMetadata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		result.AuditHumanReadable = fmt.Sprintf("error: failed to unmarshal credentials err: %v", err)
		return result
	}
	/*
		session.NewSession(&aws.Config{
			Region:      aws.String("us-west-2"),
			Credentials: credentials.NewStaticCredentials(awsCredentials.AccessKeyID, awsCredentials.AccessKeySecret, ""),
		})
	*/

	result.AuditHumanReadable = fmt.Sprintf("creating user account: ")
	return result
}

func (I IAMUserCreateResourcePostAction) Path() string {
	return ""
}
