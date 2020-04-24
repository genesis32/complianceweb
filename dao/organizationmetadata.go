package dao

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// OrganizationMetadata contains a generic dictionary of metadata for the organization.
type OrganizationMetadata map[string]interface{}

// Value is required to serialize it.
func (a OrganizationMetadata) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan is required to deserialize it.
func (a *OrganizationMetadata) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}
