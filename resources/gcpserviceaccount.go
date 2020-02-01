package resources

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/genesis32/complianceweb/dao"

	"google.golang.org/api/googleapi"

	"github.com/pkg/errors"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"

	"github.com/genesis32/complianceweb/utils"
)

type GcpServiceAccountCreateRequest struct {
	UniqueIdentifier string
}

type GcpServiceAccountCreateResponse struct {
	UniqueIdentifier string
}

type GcpServiceAccountResourcePostAction struct {
	db *sql.DB
}

func (g *GcpServiceAccountResourcePostAction) Method() string {
	return "POST"
}

func (g *GcpServiceAccountResourcePostAction) PermissionName() string {
	return "gcp.serviceaccount.write.execute"
}

type GcpServiceAcountKeyState struct {
	Name string
}

type GcpServiceAccountState struct {
	Disabled       bool
	ProjectId      string
	OrganizationID int64 `json:",string,omitempty"`
	Keys           []GcpServiceAcountKeyState
}

func (a GcpServiceAccountState) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Make the Attrs struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (a *GcpServiceAccountState) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
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
		    state = $2,
		WHERE
			external_ref = $1
	`
	_, err := db.Exec(sqlStatement, utils.GetNextUniqueId(), serviceAccountEmail, state)
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

	a := &GcpServiceAccountResourcePostAction{db: daoHandler.GetRawDatabaseHandle()}

	if credentials, ok := metadata["gcpCredentials"]; ok {
		ctx := context.Background()
		projectId := metadata["gcpProject"].(string)
		jsonBytes, err := json.Marshal(credentials)
		if err != nil {
			w.WriteHeader(500)
			result.AuditHumanReadable = fmt.Sprintf("failed: failed to unmarshal credentials err: %v", err)
			return result
		}
		serviceAccount, err := createServiceAccount(ctx, jsonBytes, projectId, req.UniqueIdentifier)
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
		initialState.ProjectId = projectId

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

func createServiceAccount(ctx context.Context, jsonCredential []byte, projectId, uniqueName string) (*iam.ServiceAccount, error) {
	var err error

	// TODO: Store and cache this somewhere.
	credentials, err := google.CredentialsFromJSON(ctx, jsonCredential, iam.CloudPlatformScope)
	client := oauth2.NewClient(context.Background(), credentials.TokenSource)

	service, err := iam.New(client)
	if err != nil {
		return nil, errors.Wrap(err, "iam.New failed")
	}

	resource := fmt.Sprintf("projects/%s", projectId)
	request := &iam.CreateServiceAccountRequest{AccountId: uniqueName, ServiceAccount: &iam.ServiceAccount{DisplayName: uniqueName}}

	serviceAccount, err := service.Projects.ServiceAccounts.Create(resource, request).Do()
	if err != nil {
		return nil, errors.Wrapf(err, "CreateServiceAccount failed")
	}
	return serviceAccount, nil
}

func createKey(ctx context.Context, jsonCredential []byte, serviceAccountEmail string) (*iam.ServiceAccountKey, error) {
	var err error
	// Make a client that relies on a service account from the db.

	credentials, err := google.CredentialsFromJSON(ctx, jsonCredential, iam.CloudPlatformScope)
	client := oauth2.NewClient(context.Background(), credentials.TokenSource)

	//	client, err := google.DefaultClient(context.Background(), iam.CloudPlatformScope)
	//	if err != nil {
	//		return nil, fmt.Errorf("google.DefaultClient: %v", err)
	//	}

	service, err := iam.New(client)
	if err != nil {
		return nil, fmt.Errorf("iam.New: %v", err)
	}

	resource := "projects/hilobit-165520/serviceAccounts/" + serviceAccountEmail
	request := &iam.CreateServiceAccountKeyRequest{}

	key, err := service.Projects.ServiceAccounts.Keys.Create(resource, request).Do()
	if err != nil {
		return nil, fmt.Errorf("Projects.ServiceAccounts.Keys.Create: %v", err)
	}
	return key, nil
}

func mapAppParameters(params OperationParameters) (dao.ResourceDaoHandler, dao.OrganizationMetadata, int64) {
	daoHandler, ok := params["resourceDao"].(dao.ResourceDaoHandler)
	if !ok {
		log.Fatal("params['resourceDao'] not a ResourceDao type")
	}

	metadata, ok := params["organizationMetadata"].(dao.OrganizationMetadata)
	if !ok {
		log.Fatal("params['organizationMetadata'] not a ResourceDao type")
	}
	organizationID, ok := params["organizationID"].(int64)
	if !ok {
		log.Fatal("params['organizationID'] not an int64 type")
	}
	return daoHandler, metadata, organizationID
}
