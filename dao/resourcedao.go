package dao

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

// ResourceDaoHandler is the separate datastore you can use in ResourceHandlers.
type ResourceDaoHandler interface {
	GetRawDatabaseHandle() (Db *sql.DB)
	Open()
}

// ResourceDao contains the pointer to the db resource.
type ResourceDao struct {
	Db *sql.DB
}

// NewResourceDaoHandler  returns a new handler backed by db.
func NewResourceDaoHandler(db *sql.DB) *ResourceDao {
	return &ResourceDao{Db: db}
}

// GetRawDatabaseHandle returns the raw database handler.
func (d *ResourceDao) GetRawDatabaseHandle() (DB *sql.DB) {
	if d.Db == nil {
		log.Fatal("database object not valid")
	}
	return d.Db
}

// Open the resource.
// TODO: For now this is the same as the one in DAO it should switch to be more configurable
func (d *ResourceDao) Open() {
	var err error

	var dbConnectionString string
	if os.Getenv("ENV") == "prod" {
		dbConnectionString = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			os.Getenv("ENTERPRISEPORTAL2_POSTGRES_PORT_5432_TCP_ADDR"),
			os.Getenv("ENTERPRISEPORTAL2_POSTGRES_PORT_5432_TCP_PORT"),
			os.Getenv("ENTERPRISEPORTAL2_POSTGRES_USER"),
			os.Getenv("ENTERPRISEPORTAL2_POSTGRES_PASSWORD"),
			os.Getenv("ENTERPRISEPORTAL2_POSTGRES_DBNAME"))
	} else {
		dbConnectionString = os.Getenv("PGSQL_CONNECTION_STRING")
		if len(dbConnectionString) == 0 {
			log.Fatal("PGSQL_CONNECTION_STRING undefined")
		}
	}
	d.Db, err = sql.Open("postgres", dbConnectionString)
	if err != nil {
		log.Fatal(err)
	}
}
