package dao

import (
	"encoding/base64"
	"log"
)

const (
	GcpAccount = "GCP"
)

type RegisteredResource struct {
	ID          int64
	DisplayName string
	InternalKey string
	Enabled     bool
}

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

type Role struct {
	ID          int64
	DisplayName string
	Permissions []*Permission
}

type Permission struct {
	ID          int64
	DisplayName string
	Value       string
}

type Setting struct {
	Key   string
	Value string
}

func (s *Setting) Base64EncodeValue(val []byte) {
	s.Value = base64.StdEncoding.EncodeToString(val)
}

func (s *Setting) Base64DecodeValue() []byte {
	ret, err := base64.StdEncoding.DecodeString(s.Value)
	if err != nil {
		log.Fatalf("cannot decode key %s: %w", s.Key, err)
	}
	return ret
}

func (o *Organization) EncodeMasterAccountCredential(cred string) {
	o.masterAccountCredential = cred
}

func (o *Organization) DecodeMasterAccountCredential() string {
	return o.masterAccountCredential
}
