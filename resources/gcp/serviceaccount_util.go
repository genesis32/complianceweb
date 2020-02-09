package gcp

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"

	"github.com/genesis32/complianceweb/resources"

	"github.com/genesis32/complianceweb/dao"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
)

type ServiceAcountKeyState struct {
	Name              string
	CreateKeyResponse *iam.ServiceAccountKey // TODO: Maybe copy this into another structure we control?
}

type ServiceAccountState struct {
	Disabled       bool
	ProjectId      string
	OrganizationID int64 `json:",string,omitempty"`
	Keys           []*ServiceAcountKeyState
}

func (a ServiceAccountState) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Make the Attrs struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (a *ServiceAccountState) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

func createServiceAccount(ctx context.Context, jsonCredential []byte, projectId, uniqueName string, roles []string) (*iam.ServiceAccount, error) {
	var err error

	// TODO: Store and cache this somewhere.
	credentials, err := google.CredentialsFromJSON(ctx, jsonCredential, iam.CloudPlatformScope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read credentials from json")
	}
	client := oauth2.NewClient(ctx, credentials.TokenSource)

	iamService, err := iam.New(client)
	if err != nil {
		return nil, errors.Wrap(err, "iam.New failed")
	}

	resource := fmt.Sprintf("projects/%s", projectId)
	request := &iam.CreateServiceAccountRequest{AccountId: uniqueName, ServiceAccount: &iam.ServiceAccount{DisplayName: uniqueName}}

	serviceAccount, err := iamService.Projects.ServiceAccounts.Create(resource, request).Do()
	if err != nil {
		return nil, errors.Wrapf(err, "CreateServiceAccount failed")
	}

	cloudresourcemanagerService, err := cloudresourcemanager.New(client)
	if err != nil {
		log.Fatal(err)
	}

	rb := &cloudresourcemanager.GetIamPolicyRequest{
		// TODO: Add desired fields of the request body.
	}

	iamPolicyResp, err := cloudresourcemanagerService.Projects.GetIamPolicy(projectId, rb).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Validate me
	for _, role := range roles {
		iamPolicyResp.Bindings = append(iamPolicyResp.Bindings, &cloudresourcemanager.Binding{Members: []string{"serviceAccount:" + serviceAccount.Email}, Role: role})
	}

	sb := &cloudresourcemanager.SetIamPolicyRequest{
		Policy: iamPolicyResp,
	}
	_, err = cloudresourcemanagerService.Projects.SetIamPolicy(projectId, sb).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	return serviceAccount, nil
}

func createServiceAccountKey(ctx context.Context, jsonCredential []byte, serviceAccountEmail string) (*iam.ServiceAccountKey, error) {
	var err error
	// Make a client that relies on a service account from the db.

	credentials, err := google.CredentialsFromJSON(ctx, jsonCredential, iam.CloudPlatformScope)
	client := oauth2.NewClient(context.Background(), credentials.TokenSource)

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

func mapAppParameters(params resources.OperationParameters) (dao.ResourceDaoHandler, dao.OrganizationMetadata, int64) {
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
