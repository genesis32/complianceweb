package dao

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type OrganizationMetadata map[string]interface{}

func (a OrganizationMetadata) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Make the Attrs struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (a *OrganizationMetadata) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}
