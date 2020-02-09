package gcp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/genesis32/complianceweb/resources"

	"google.golang.org/api/iam/v1"
)

type GcpServiceAccountKeyGetResponse struct {
	Key *iam.ServiceAccountKey
}

type GcpServiceAccountResourceKeyGetAction struct {
	db *sql.DB
}

func (g GcpServiceAccountResourceKeyGetAction) Path() string {
	return ""
}

func (g GcpServiceAccountResourceKeyGetAction) Name() string {
	return "Gcp Service Account List"
}

func (g GcpServiceAccountResourceKeyGetAction) InternalKey() string {
	return "gcp.serviceaccount.keys"
}

func (g GcpServiceAccountResourceKeyGetAction) Method() string {
	return "GET"
}

func (g GcpServiceAccountResourceKeyGetAction) PermissionName() string {
	return "gcp.serviceaccount.read.execute"
}

func (g GcpServiceAccountResourceKeyGetAction) Execute(w http.ResponseWriter, r *http.Request, params resources.OperationParameters) *resources.OperationResult {

	daoHandler, _, _ := mapAppParameters(params)

	a := &GcpServiceAccountResourceKeyGetAction{db: daoHandler.GetRawDatabaseHandle()}

	name, ok := r.URL.Query()["name"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return nil
	}

	result := resources.NewOperationResult()

	serviceAccountRef, state := retrieveStateForKey(a.db, name[0])
	if serviceAccountRef == "" {
		w.WriteHeader(http.StatusNotFound)
		return nil
	}

	for _, key := range state.Keys {
		if key.Name == name[0] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(key)
			result.AuditMetadata["Name"] = key.Name
			result.AuditHumanReadable = fmt.Sprintf("retrieved key material for service account: %s", serviceAccountRef)
			return result
		}
	}
	log.Fatal(errors.New("key came back in a query but did not exist in array"))
	return nil
}

func retrieveStateForKey(db *sql.DB, keyID string) (string, *GcpServiceAccountState) {
	// TODO: Validate KeyID format, also this is freaking horrible FIX IT
	m := make(map[string]string)
	m["name"] = keyID
	bytes, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}

	sqlStatement := `
		SELECT 
		external_ref, state 
		FROM 
			resource_gcpserviceaccount 
		WHERE 
			jsonb_path_exists(state, '$.Keys[*].Name ? (@ == $name)', $1);
	`

	var serviceAccount string
	ret := GcpServiceAccountState{}

	row := db.QueryRow(sqlStatement, string(bytes))
	err = row.Scan(&serviceAccount, &ret)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		log.Fatal(err)
	}
	return serviceAccount, &ret
}
