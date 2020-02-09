package resources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"google.golang.org/api/googleapi"

	"github.com/genesis32/complianceweb/utils"
)

type GcpServiceAccountCreateRequest struct {
	UniqueIdentifier string
	ProjectID        string
	Roles            []string
}

type GcpServiceAccountCreateResponse struct {
	UniqueIdentifier string
}

type GcpServiceAccountResourcePostAction struct{ db *sql.DB }

func (g *GcpServiceAccountResourcePostAction) Path() string {
	return ""
}

func (g *GcpServiceAccountResourcePostAction) Method() string {
	return "POST"
}

func (g *GcpServiceAccountResourcePostAction) PermissionName() string {
	return "gcp.serviceaccount.write.execute"
}

func (g *GcpServiceAccountResourcePostAction) createServiceAccountRecord(emailAddress string, state GcpServiceAccountState) {
	var err error
	jsonBytes, err := json.Marshal(state)
	if err != nil {
		log.Fatal(err)
	}

	sqlStatement := `
		INSERT INTO resource_gcpserviceaccount
			(id, external_ref, state)
		VALUES 
			($1, $2, $3)
	`
	_, err = g.db.Exec(sqlStatement, utils.GetNextUniqueId(), emailAddress, string(jsonBytes))
	if err != nil {
		log.Fatal(err)
	}
}

/*
	params["organizationID"] = organizationID
	params["organizationMetadata"] = metadata
	params["resourceDao"] = s.ResourceDao
	params["userInfo"] = userInfo
*/
func (g *GcpServiceAccountResourcePostAction) Execute(w http.ResponseWriter, r *http.Request, params OperationParameters) *OperationResult {

	daoHandler, metadata, organizationID := mapAppParameters(params)

	result := newOperationResult()

	var req GcpServiceAccountCreateRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		result.AuditHumanReadable = fmt.Sprintf("error: failed to unmarshal request err: %v", err)
		return result
	}

	// TODO: Check that this service account exists in the project.

	a := &GcpServiceAccountResourcePostAction{db: daoHandler.GetRawDatabaseHandle()}

	if credentials, ok := metadata["gcpCredentials"]; ok {
		ctx := context.Background()
		jsonBytes, err := json.Marshal(credentials)
		if err != nil {
			w.WriteHeader(500)
			result.AuditHumanReadable = fmt.Sprintf("failed: failed to unmarshal credentials err: %v", err)
			return result
		}
		serviceAccount, err := createServiceAccount(ctx, jsonBytes, req.ProjectID, req.UniqueIdentifier, req.Roles)
		if err != nil {
			if e, ok := err.(*googleapi.Error); ok {
				w.WriteHeader(e.Code)
				w.Write([]byte(e.Message))
				result.AuditHumanReadable = fmt.Sprintf("failed: failed to create gcp service account googleapi error: %v", err)
				return result
			} else {
				w.WriteHeader(500)
				result.AuditHumanReadable = fmt.Sprintf("failed: failed to create gcp service account other error: %v", err)
				return result
			}
		}

		initialState := GcpServiceAccountState{}
		initialState.Disabled = serviceAccount.Disabled
		initialState.OrganizationID = organizationID
		initialState.ProjectId = req.ProjectID

		a.createServiceAccountRecord(serviceAccount.Email, initialState)
		w.WriteHeader(200)
		result.AuditHumanReadable = fmt.Sprintf("success: created gcp service account: %s", serviceAccount.Email)
		return result

	} else {
		w.WriteHeader(404)
		result.AuditHumanReadable = fmt.Sprintf("failed: could not find gcp credentials")
		return result
	}
}

func (g *GcpServiceAccountResourcePostAction) Name() string {
	return "GCP Service Account Manager Create"
}

func (g *GcpServiceAccountResourcePostAction) InternalKey() string {
	return "gcp.serviceaccount"
}

func retrieveState(db *sql.DB, serviceAccountEmail string) *GcpServiceAccountState {
	sqlStatement := `
		SELECT
			state
		FROM
			resource_gcpserviceaccount
		WHERE
			external_ref = $1
	`

	ret := GcpServiceAccountState{}
	row := db.QueryRow(sqlStatement, serviceAccountEmail)
	err := row.Scan(&ret)
	if err != nil {
		log.Fatal(err)
	}
	return &ret
}

func updateState(db *sql.DB, serviceAccountEmail string, state *GcpServiceAccountState) {
	sqlStatement := `
		UPDATE 
			resource_gcpserviceaccount
		SET
		    state = $2
		WHERE
			external_ref = $1
	`
	_, err := db.Exec(sqlStatement, serviceAccountEmail, state)
	if err != nil {
		log.Fatal(err)
	}
}
