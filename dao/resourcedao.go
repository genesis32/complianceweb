package dao

import (
	"database/sql"
	"log"
	"os"
)

type ResourceDaoHandler interface {
	Open()
	Audit()
}

type ResourceDao struct {
	Db *sql.DB
}

func (d *ResourceDao) Audit() {
	panic("implement me")
}

func NewResourceDaoHandler(db *sql.DB) *ResourceDao {
	return &ResourceDao{Db: db}
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
