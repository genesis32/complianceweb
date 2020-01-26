package dao

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/genesis32/complianceweb/utils"
)

type AuditRecord struct {
	ID                 int64
	CreatedTimestamp   time.Time
	OrganizationUserID int64
	OrganizationID     int64
	InternalKey        string
	Method             string
	Metadata           map[string]interface{}
	HumanReadable      string
}

func NewAuditRecord(internalKey, method string) *AuditRecord {
	return &AuditRecord{
		ID:               utils.GetNextUniqueId(),
		CreatedTimestamp: time.Now(),
		InternalKey:      internalKey,
		Method:           method,
	}
}

type ResourceDaoHandler interface {
	GetRawDatabaseHandle() (Db *sql.DB)
	Open()
	Audit(record *AuditRecord)
}

type ResourceDao struct {
	Db *sql.DB
}

func NewResourceDaoHandler(db *sql.DB) *ResourceDao {
	return &ResourceDao{Db: db}
}

func (d *ResourceDao) GetRawDatabaseHandle() (DB *sql.DB) {
	if d.Db == nil {
		log.Fatalf("database object not valid")
	}
	return d.Db
}

func (d *ResourceDao) Open() {
	var err error

	dbConnectionString := os.Getenv("PGSQL_CONNECTION_STRING")
	if len(dbConnectionString) == 0 {
		panic("PGSQL_CONNECTION_STRING undefined")
	}
	d.Db, err = sql.Open("postgres", dbConnectionString)
	if err != nil {
		log.Fatal(err)
	}
}

func (d *ResourceDao) Audit(record *AuditRecord) {

}
