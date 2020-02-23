package aws

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/genesis32/complianceweb/utils"

	"github.com/genesis32/complianceweb/resources"
	"gopkg.in/validator.v2"
)

type IAMUserCreateResourcePostAction struct {
	db *sql.DB
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

func (g *IAMUserCreateResourcePostAction) createRecord(identifier string, state iamUserState) int64 {
	var err error
	jsonBytes, err := json.Marshal(state)
	if err != nil {
		log.Fatal(err)
	}

	sqlStatement := `
		INSERT INTO resource_awsiam
			(id, external_ref, state)
		VALUES 
			($1, $2, $3)
	`
	ret := utils.GetNextUniqueId()
	_, err = g.db.Exec(sqlStatement, ret, identifier, string(jsonBytes))
	if err != nil {
		log.Fatal(err)
	}
	return ret
}

func (g *IAMUserCreateResourcePostAction) Execute(w http.ResponseWriter, r *http.Request, params resources.OperationParameters) *resources.OperationResult {
	result := resources.NewOperationResult()

	daoHandler, metadata, _, theUser := resources.MapAppParameters(params)

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

	a := &IAMUserCreateResourcePostAction{db: daoHandler.GetRawDatabaseHandle()}
	state := iamUserState{CreateRequest: req, State: UserStateCreatedNotApproved, UserIDCreatedBy: theUser.ID}

	recordID := a.createRecord(req.UserName, state)
	resp := IAMUserCreateResponse{ID: recordID, UserName: req.UserName}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if errs := json.NewEncoder(w).Encode(&resp); errs != nil {
		result.AuditHumanReadable = fmt.Sprintf("error encoding response: %s", errs.Error())
		return result
	}

	result.AuditHumanReadable = fmt.Sprintf("created: %+v. waiting for approval", req.UserName)

	return result
}

func (I IAMUserCreateResourcePostAction) Path() string {
	return ""
}
