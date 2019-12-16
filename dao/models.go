package dao

const (
	GcpAccount = "GCP"
)

type ServiceAccountCredentials struct {
	OwningOrganizationID int64
	Type                 string
	Credentials          map[string]interface{}
	RawCredentials       []byte
}

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
	masterAccountCredential string // TODO: Break this out later
	Path                    string
	Users                   []*OrganizationUser
}

type OrganizationUser struct {
	ID            int64
	DisplayName   string
	Organizations []int64
	Active        bool
}

func (o *Organization) EncodeMasterAccountCredential(cred string) {
	o.masterAccountCredential = cred
}

func (o *Organization) DecodeMasterAccountCredential() string {
	return o.masterAccountCredential
}
