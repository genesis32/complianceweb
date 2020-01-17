package dao

import (
	"encoding/base64"
	"log"
)

type RegisteredResource struct {
	ID          int64
	DisplayName string
	InternalKey string
	Enabled     bool
}

type Organization struct {
	ID          int64
	DisplayName string
	Path        string
	Users       []*OrganizationUser
}

type OrganizationUser struct {
	ID            int64
	DisplayName   string
	Organizations []int64
	Active        bool
	UserRoles     UserRoleStore
}

type UserRole struct {
	OrganizationID int64
	RoleNames      []string
}

type Role struct {
	ID          int64
	DisplayName string
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
