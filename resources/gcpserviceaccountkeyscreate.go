package resources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

func (g *GcpServiceAccountResourceKeyPostAction) Path() string {
	return ""
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
	daoHandler, metadata, _ := mapAppParameters(params)

	a := &GcpServiceAccountResourcePostAction{db: daoHandler.GetRawDatabaseHandle()}
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
		if err != nil {
			w.WriteHeader(500)
			result.AuditHumanReadable = fmt.Sprintf("failed: failed to create key err: %v", err)
			return result
		}

		state := retrieveState(a.db, req.GcpEmailIdentifier)
		newKey := &GcpServiceAcountKeyState{Name: serviceAccountKey.Name, CreateKeyResponse: serviceAccountKey}
		state.Keys = append(state.Keys, newKey)
		updateState(a.db, req.GcpEmailIdentifier, state)
		result.AuditHumanReadable = fmt.Sprintf("created new key for service account: %s", req.GcpEmailIdentifier)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newKey)

	}

	return result
}
