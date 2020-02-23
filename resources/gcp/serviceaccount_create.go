package gcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/genesis32/complianceweb/dao"

	"github.com/genesis32/complianceweb/resources"

	"google.golang.org/api/googleapi"

	"github.com/genesis32/complianceweb/utils"
)

type ServiceAccountResourcePostAction struct{ db *sql.DB }

func (g *ServiceAccountResourcePostAction) Path() string {
	return ""
}

func (g *ServiceAccountResourcePostAction) Method() string {
	return "POST"
}

func (g *ServiceAccountResourcePostAction) PermissionName() string {
	return "gcp.serviceaccount.write.execute"
}

func (g *ServiceAccountResourcePostAction) RequiredMetadata() []string {
	return []string{"gcpCredentials"}
}

func (g *ServiceAccountResourcePostAction) createServiceAccountRecord(emailAddress string, state ServiceAccountState) {
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
func (g *ServiceAccountResourcePostAction) Execute(w http.ResponseWriter, r *http.Request, params resources.OperationParameters) *resources.OperationResult {

	daoHandler, metadataBytes, organizationID, _ := resources.MapAppParameters(params)

	result := resources.NewOperationResult()

	var req ServiceAccountCreateRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		result.AuditHumanReadable = fmt.Sprintf("error: failed to unmarshal request err: %v", err)
		return result
	}

	// TODO: Check that this service account exists in the project.

	a := &ServiceAccountResourcePostAction{db: daoHandler.GetRawDatabaseHandle()}

	var metadata dao.OrganizationMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		log.Fatal(err)
	}

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

		initialState := ServiceAccountState{}
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

func (g *ServiceAccountResourcePostAction) Name() string {
	return "GCP Service Account Manager Create"
}

func (g *ServiceAccountResourcePostAction) InternalKey() string {
	return "gcp.serviceaccount"
}
