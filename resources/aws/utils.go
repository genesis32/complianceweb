package aws

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"log"
)

const (
	UserStateCreatedNotApproved = 1
	UserStateApproved           = 2
)

type iamUserState struct {
	State           int
	UserIDCreatedBy int64
	CreateRequest   IAMUserCreateRequest
}

func (a iamUserState) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Make the Attrs struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (a *iamUserState) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

type resourceMetadata struct {
	AWSCredentials awsCredentials
}
type awsCredentials struct {
	AccessKeyID     string
	AccessKeySecret string
}

func retrieveState(db *sql.DB, ID int64) *iamUserState {
	sqlStatement := `
		SELECT
			state
		FROM
			resource_awsiam
		WHERE
			id = $1
	`

	ret := iamUserState{}
	row := db.QueryRow(sqlStatement, ID)
	err := row.Scan(&ret)
	if err != nil {
		log.Fatal(err)
	}
	return &ret
}

func updateState(db *sql.DB, ID int64, state *iamUserState) {
	sqlStatement := `
		UPDATE 
			resource_awsiam
		SET
		    state = $2
		WHERE
			ID = $1
	`
	_, err := db.Exec(sqlStatement, ID, state)
	if err != nil {
		log.Fatal(err)
	}
}
