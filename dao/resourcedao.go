package dao

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

type ResourceDaoHandler interface {
	GetRawDatabaseHandle() (Db *sql.DB)
	Open()
}

type ResourceDao struct {
	Db *sql.DB
}

func NewResourceDaoHandler(db *sql.DB) *ResourceDao {
	return &ResourceDao{Db: db}
}

func (d *ResourceDao) GetRawDatabaseHandle() (DB *sql.DB) {
	if d.Db == nil {
		log.Fatal("database object not valid")
	}
	return d.Db
}

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
