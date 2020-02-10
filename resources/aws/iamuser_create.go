package aws

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/genesis32/complianceweb/resources"
	"gopkg.in/validator.v2"
)

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

	var req IAMUserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		result.AuditHumanReadable = fmt.Sprintf("error: failed to unmarshal request err: %v", err)
		return result
	}

	if errs := validator.Validate(req); errs != nil {
		http.Error(w, errs.Error(), http.StatusBadRequest)
		result.AuditHumanReadable = fmt.Sprintf("error: failed to validate request err: %v", errs)
		return result
	}

	var resourceMetadata resourceMetadata
	if err := json.Unmarshal(metadata, &resourceMetadata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		result.AuditHumanReadable = fmt.Sprintf("error: failed to unmarshal credentials err: %v", err)
		return result
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
		Credentials: credentials.NewStaticCredentials(resourceMetadata.AWSCredentials.AccessKeyID,
			resourceMetadata.AWSCredentials.AccessKeySecret, ""),
	})

	// Create a IAM service client.
	svc := iam.New(sess)

	createUserResult, err := svc.CreateUser(&iam.CreateUserInput{
		UserName: &req.UserName,
	})

	if err != nil {
		result.AuditHumanReadable = fmt.Sprintf("error creating user account: ")
		return result
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if errs := json.NewEncoder(w).Encode(&createUserResult); errs != nil {
		result.AuditHumanReadable = fmt.Sprintf("error encoding response: %s", errs.Error())
		return result
	}

	result.AuditHumanReadable = fmt.Sprintf("created user account: %+v", createUserResult)

	return result
}

func (I IAMUserCreateResourcePostAction) Path() string {
	return ""
}
