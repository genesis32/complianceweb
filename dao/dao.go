package dao

import (
	"database/sql"
	"log"
	"math/rand"

	_ "github.com/lib/pq"
)

type DaoHandler interface {
	Open() error
	Close() error
	TrySelect()
	CreateOrUpdateUser(name, subject string)
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

func (d *Dao) CreateOrUpdateUser(name, subject string) {
	sqlStatement := `
	INSERT INTO ouser (id, display_name, credential_value, last_login_timestamp) 
	VALUES ($1, $2, $3, NOW())
	ON CONFLICT (credential_value) DO UPDATE SET last_login_timestamp = NOW()
	`
	_, err := d.Db.Exec(sqlStatement, rand.Int63(), name, subject)
	if err != nil {
		panic(err)
	}
}

func (d *Dao) TrySelect() {
	sqlStatement := `SELECT id FROM organization WHERE display_name='baz'`
	row := d.Db.QueryRow(sqlStatement)
	var out int
	err := row.Scan(&out)
	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}
	log.Printf("row: %d", out)
}

func (d *Dao) Close() error {
	err := d.Db.Close()
	return err
}
