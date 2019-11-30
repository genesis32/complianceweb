package server

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
)

func createKey(w io.Writer, serviceAccountEmail string) (*iam.ServiceAccountKey, error) {

	// Make a client that relies on a service account from the db.

	client, err := google.DefaultClient(context.Background(), iam.CloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("google.DefaultClient: %v", err)
	}
	service, err := iam.New(client)
	if err != nil {
		return nil, fmt.Errorf("iam.New: %v", err)
	}

	resource := "projects/-/serviceAccounts/" + serviceAccountEmail
	request := &iam.CreateServiceAccountKeyRequest{}
	key, err := service.Projects.ServiceAccounts.Keys.Create(resource, request).Do()
	if err != nil {
		return nil, fmt.Errorf("Projects.ServiceAccounts.Keys.Create: %v", err)
	}
	fmt.Fprintf(w, "Created key: %v", key.Name)
	return key, nil
}
