package dao

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type DaoHandler interface {
	Open() error
	TrySelect()
}

type Dao struct {
	Db *sql.DB
}

func NewDaoHandler() DaoHandler {
	return &Dao{Db: nil}
}

func (d *Dao) Open() error {
	var err error
	d.Db, err = sql.Open("postgres", "user=dmassey dbname=dmassey sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (d *Dao) TrySelect() {
	sqlStatement := `SELECT id FROM organization WHERE display_name='baz'`
	row := d.Db.QueryRow(sqlStatement)
	var out int
	err := row.Scan(&out)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("row: %d", out)
}
