package dao

import (
	"encoding/base64"
	"log"
)

// RegisteredResource is the entity which determines which resources the app is protecting
type RegisteredResource struct {
	ID          int64
	DisplayName string
	InternalKey string
	Enabled     bool
}

// Organization is used as part of a tree to determine permissions
type Organization struct {
	ID          int64
	DisplayName string
	Path        string
	Users       []*OrganizationUser
}

// OrganizationUser is the user that is part of an organization
// it also contains the explicit roles a user has in each suborganization.
type OrganizationUser struct {
	ID            int64
	DisplayName   string
	Organizations []int64
	CurrentState  int
	UserRoles     UserRoleStore
}

// UserRole describes the roles a user has in an organization
type UserRole struct {
	OrganizationID int64
	RoleNames      []string
}

// Role is just a collection of permissions
type Role struct {
	ID          int64
	DisplayName string
}

// Setting contains just a key value mapping of settings for the app
type Setting struct {
	Key   string
	Value string
}

// Base64EncodeValue just base64 encodes the settings value.
func (s *Setting) Base64EncodeValue(val []byte) {
	s.Value = base64.StdEncoding.EncodeToString(val)
}

// Base64DecodeValue just base64 decodes the settings value.
func (s *Setting) Base64DecodeValue() []byte {
	ret, err := base64.StdEncoding.DecodeString(s.Value)
	if err != nil {
		log.Fatalf("cannot decode key %s: %v", s.Key, err)
	}
	return ret
}
