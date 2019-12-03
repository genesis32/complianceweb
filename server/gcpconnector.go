package server

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
)

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
