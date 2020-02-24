package aws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/genesis32/complianceweb/resources"
	"gopkg.in/validator.v2"
)

type IAMUserApproveResourcePostAction struct {
}

func (I IAMUserApproveResourcePostAction) RequiredMetadata() []string {
	return []string{"awsCredentials"}
}

func (I IAMUserApproveResourcePostAction) Name() string {
	return "AWS IAM User Approve"
}

func (I IAMUserApproveResourcePostAction) InternalKey() string {
	return "aws.iam.user"
}

func (I IAMUserApproveResourcePostAction) Method() string {
	return "POST"
}

func (I IAMUserApproveResourcePostAction) PermissionName() string {
	return "aws.iam.user.create.execute"
}

func (I IAMUserApproveResourcePostAction) Execute(w http.ResponseWriter, r *http.Request, params resources.OperationParameters) *resources.OperationResult {
	result := resources.NewOperationResult()

	daoHandler, metadata, _, theUser := resources.MapAppParameters(params)

	var req IAMUserApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		result.AuditHumanReadable = fmt.Sprintf("error: failed to unmarshal request err: %v", err)
		return result
	}

	if err := validator.Validate(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		result.AuditHumanReadable = fmt.Sprintf("error: failed to validate request err: %v", err)
		return result
	}

	var resourceMetadata resourceMetadata
	if err := json.Unmarshal(metadata, &resourceMetadata); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		result.AuditHumanReadable = fmt.Sprintf("error: failed to unmarshal credentials err: %v", err)
		return result
	}

	theState := retrieveState(daoHandler.GetRawDatabaseHandle(), req.ID)
	if theState == nil {
		w.WriteHeader(http.StatusNotFound)
		result.AuditHumanReadable = fmt.Sprintf("user: %d resource: %d not found", theUser.ID, req.ID)
		return result
	}

	if theUser.ID == theState.UserIDCreatedBy {
		w.WriteHeader(http.StatusUnauthorized)
		result.AuditHumanReadable = fmt.Sprintf("user: %d cannot approve their own request: %d", theUser.ID, req.ID)
		return result
	}

	resp := IAMUserApproveResponse{Approved: true, ID: req.ID}
	if false {
		if sess, err := session.NewSession(&aws.Config{
			Region: aws.String("us-west-2"),
			Credentials: credentials.NewStaticCredentials(resourceMetadata.AWSCredentials.AccessKeyID,
				resourceMetadata.AWSCredentials.AccessKeySecret, ""),
		}); err != nil {
			log.Fatal(err)
		} else {
			svc := iam.New(sess)
			p := &iam.CreateUserInput{
				UserName: &req.UserName,
			}
			// Create a IAM service client.
			if resp.CreateUserOutput, err = svc.CreateUser(p); err != nil {
				result.AuditHumanReadable = fmt.Sprintf("error creating user account: ")
				return result
			}
		}
	}
	theState.State = UserStateApproved
	updateState(daoHandler.GetRawDatabaseHandle(), req.ID, theState)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		log.Fatal(err)
	}

	result.AuditHumanReadable = fmt.Sprintf("approved account id: %d name:%s", req.ID, req.UserName)
	return result
}

func (I IAMUserApproveResourcePostAction) Path() string {
	return "approve"
}
