package dao

import (
	"database/sql"
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
