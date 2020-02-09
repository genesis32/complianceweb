package gcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/genesis32/complianceweb/resources"
)

type ServiceAccountKeyCreateRequest struct {
	GcpEmailIdentifier string
}

type ServiceAccountKeyCreateResponse struct {
	UniqueIdentifier string
}

type ServiceAccountResourceKeyPostAction struct {
	db *sql.DB
}

func (g *ServiceAccountResourceKeyPostAction) Path() string {
	return ""
}

func (g ServiceAccountResourceKeyPostAction) Name() string {
	return "Create GCP Service Account Key"
}

func (g ServiceAccountResourceKeyPostAction) InternalKey() string {
	return "gcp.serviceaccount.keys"
}

func (g ServiceAccountResourceKeyPostAction) Method() string {
	return "POST"
}

func (g ServiceAccountResourceKeyPostAction) PermissionName() string {
	return "gcp.serviceaccount.write.execute"
}

func (g ServiceAccountResourceKeyPostAction) Execute(w http.ResponseWriter, r *http.Request, params resources.OperationParameters) *resources.OperationResult {
	daoHandler, metadata, _ := mapAppParameters(params)

	a := &ServiceAccountResourcePostAction{db: daoHandler.GetRawDatabaseHandle()}
	result := resources.NewOperationResult()

	var req ServiceAccountKeyCreateRequest
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
		serviceAccountKey, err := createServiceAccountKey(ctx, jsonBytes, req.GcpEmailIdentifier)
		if err != nil {
			w.WriteHeader(500)
			result.AuditHumanReadable = fmt.Sprintf("failed: failed to create key err: %v", err)
			return result
		}

		state := retrieveState(a.db, req.GcpEmailIdentifier)
		newKey := &ServiceAcountKeyState{Name: serviceAccountKey.Name, CreateKeyResponse: serviceAccountKey}
		state.Keys = append(state.Keys, newKey)
		updateState(a.db, req.GcpEmailIdentifier, state)
		result.AuditHumanReadable = fmt.Sprintf("created new key for service account: %s", req.GcpEmailIdentifier)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newKey)
	}

	return result
}
