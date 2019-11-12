package dao

const (
	GcpAccount = "GCP"
)

type User struct {
	ID                    int64
	DisplayName           string
	CredentialValue       string
	OwningOrganizationIDs []int64
}

type Organization struct {
	ID                      int64
	DisplayName             string
	MasterAccountType       string
	masterAccountCredential string
}

type OrganizationUser struct {
	ID          int64
	DisplayName string
}

func (o *Organization) EncodeMasterAccountCredential(cred string) {
	o.masterAccountCredential = cred
}

func (o *Organization) DecodeMasterAccountCredential() string {
	return o.masterAccountCredential
}
