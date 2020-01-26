package resources

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"

	"github.com/genesis32/complianceweb/utils"

	"github.com/genesis32/complianceweb/dao"
)

type GcpServiceAccountResourcePostAction struct {
	db *sql.DB
}

func (g *GcpServiceAccountResourcePostAction) Method() string {
	return "POST"
}

func (g *GcpServiceAccountResourcePostAction) PermissionName() string {
	return "gcp.serviceaccount.write.execute"
}

func (g *GcpServiceAccountResourcePostAction) createDbEntry() {
	sqlStatement := `
		INSERT INTO resource_gcpserviceaccount
			(id, state)
		VALUES 
			($1, '{}')
	`
	_, err := g.db.Exec(sqlStatement, utils.GetNextUniqueId())
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
	daoHandler, ok := params["resourceDao"].(dao.ResourceDaoHandler)
	if !ok {
		log.Fatal("params['resourceDao'] not a ResourceDao type")
	}

	metadata, ok := params["organizationMetadata"].(dao.OrganizationMetadata)
	if !ok {
		log.Fatal("params['organizationMetadata'] not a ResourceDao type")
	}
	log.Printf("gcp.serviceaccount post")

	a := &GcpServiceAccountResourcePostAction{db: daoHandler.GetRawDatabaseHandle()}

	if credentials, ok := metadata["gcpCredential"]; ok {
		b, ok0 := credentials.(string)
		if ok0 {
			ctx := context.Background()
			createServiceAccount(ctx, []byte(b), "accountId")
			a.createDbEntry()
		}
	}

	w.WriteHeader(200)

	result := newOperationResult()
	result.AuditHumanReadable = fmt.Sprintf("This went ok %v", metadata)

	return result
}

func (g *GcpServiceAccountResourcePostAction) Name() string {
	return "GCP Service Account Manager Create"
}

func (g *GcpServiceAccountResourcePostAction) InternalKey() string {
	return "gcp.serviceaccount"
}

func createServiceAccount(ctx context.Context, jsonCredential []byte, serviceAccountID string) (*iam.ServiceAccount, error) {
	var err error

	// TODO: Store and cache this somewhere.
	credentials, err := google.CredentialsFromJSON(ctx, jsonCredential, iam.CloudPlatformScope)
	client := oauth2.NewClient(context.Background(), credentials.TokenSource)

	service, err := iam.New(client)
	if err != nil {
		return nil, fmt.Errorf("iam.New: %v", err)
	}

	resource := "projects/hilobit-165520"
	request := &iam.CreateServiceAccountRequest{AccountId: serviceAccountID, ServiceAccount: &iam.ServiceAccount{DisplayName: serviceAccountID}}

	key, err := service.Projects.ServiceAccounts.Create(resource, request).Do()
	if err != nil {
		return nil, fmt.Errorf("Projects.ServiceAccounts.Keys.Create: %v", err)
	}
	return key, nil
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
