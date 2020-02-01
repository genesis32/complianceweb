package resources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type GcpServiceAccountKeyCreateRequest struct {
	GcpEmailIdentifier string
}

type GcpServiceAccountKeyCreateResponse struct {
	UniqueIdentifier string
}

type GcpServiceAccountResourceKeyPostAction struct {
	db *sql.DB
}

func (g GcpServiceAccountResourceKeyPostAction) Name() string {
	return "Create GCP Service Account Key"
}

func (g GcpServiceAccountResourceKeyPostAction) InternalKey() string {
	return "gcp.serviceaccount.keys"
}

func (g GcpServiceAccountResourceKeyPostAction) Method() string {
	return "POST"
}

func (g GcpServiceAccountResourceKeyPostAction) PermissionName() string {
	return "gcp.serviceaccount.write.execute"
}

func (g GcpServiceAccountResourceKeyPostAction) Execute(w http.ResponseWriter, r *http.Request, params OperationParameters) *OperationResult {
	_, metadata, _ := mapAppParameters(params)

	result := newOperationResult()

	var req GcpServiceAccountKeyCreateRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		result.AuditHumanReadable = fmt.Sprintf("error: failed to unmarshal request err: %v", err)
		return result
	}

	if credentials, ok := metadata["gcpCredentials"]; ok {
		ctx := context.Background()
		jsonBytes, err := json.Marshal(credentials)
		if err != nil {
			w.WriteHeader(500)
			result.AuditHumanReadable = fmt.Sprintf("failed: failed to unmarshal credentials err: %v", err)
			return result
		}
		serviceAccountKey, err := createKey(ctx, jsonBytes, req.GcpEmailIdentifier)
		state := retrieveState(g.db, req.GcpEmailIdentifier)
		newKey := GcpServiceAcountKeyState{Name: serviceAccountKey.Name}
		state.Keys = append(state.Keys, newKey)
		updateState(g.db, req.GcpEmailIdentifier, state)
		log.Printf("%s", serviceAccountKey.Name)
	}

	return result
}
