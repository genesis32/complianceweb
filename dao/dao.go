package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type DaoHandler interface {
	Open() error
	Close() error
	TrySelect()
	GetNextUniqueId() int64
	CreateOrganization(*Organization) error
	CreateInviteForUser(organizationId int64, name string) (string, error)
	LoadUserFromInviteCode(inviteCode string) (*OrganizationUser, error)
	InitUserFromInviteCode(inviteCode, idpAuthCredential string) (bool, error)
	LogUserIn(idpAuthCredential string) (*OrganizationUser, error)
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

func (d *Dao) GetNextUniqueId() int64 {
	return rand.Int63()
}

func (d *Dao) LogUserIn(idpAuthCredential string) (*OrganizationUser, error) {
	sqlStatement := `SELECT id, display_name, organizations FROM organization_user WHERE idp_type = 'AUTH0' AND idp_credential_value=$1 AND current_state=1`
	var orgUser OrganizationUser

	row := d.Db.QueryRow(sqlStatement, idpAuthCredential)
	err := row.Scan(&orgUser.ID, &orgUser.DisplayName, pq.Array(orgUser.Organizations))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error loading user from credential: %w", err)
	}

	return &orgUser, nil
}

func (d *Dao) LoadUserFromInviteCode(inviteCode string) (*OrganizationUser, error) {
	sqlStatement := `SELECT id, display_name FROM organization_user WHERE invite_code=$1 AND current_state=0`
	var orgUser OrganizationUser

	row := d.Db.QueryRow(sqlStatement, inviteCode)
	err := row.Scan(&orgUser.ID, &orgUser.DisplayName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error loading user from invite code %v: %w", inviteCode, err)
	}

	return &orgUser, nil
}

func (d *Dao) CreateInviteForUser(organizationId int64, name string) (string, error) {
	inviteCode := fmt.Sprintf("%d", d.GetNextUniqueId())
	orgUserID := fmt.Sprintf("%d", d.GetNextUniqueId())

	sqlStatement := `
		INSERT INTO organization_user (id, display_name, organizations, invite_code, created_timestamp, current_state)
		VALUES ($1, $2, $3, $4, $5, 0)
	`
	_, err := d.Db.Exec(sqlStatement, orgUserID, name, fmt.Sprintf("{%d}", organizationId), inviteCode, "NOW()")
	if err != nil {
		panic(err)
	}
	return inviteCode, nil
}

func (d *Dao) CreateOrganization(org *Organization) error {
	sqlStatement := `
	INSERT INTO organization (id, display_name, master_account_type, master_account_credential) 
	VALUES ($1, $2, $3, $4)
	`
	_, err := d.Db.Exec(sqlStatement, org.ID, org.DisplayName, GcpAccount, org.masterAccountCredential)
	if err != nil {
		panic(err)
	}
	return nil
}

func (d *Dao) InitUserFromInviteCode(inviteCode, idpAuthCredential string) (bool, error) {
	sqlStatement := `
	UPDATE 
		organization_user 
    SET
		idp_type = 'AUTH0',
	    idp_credential_value = $1,
	    current_state = 1 
	WHERE
		invite_code = $2 AND current_state=0
	`
	_, err := d.Db.Exec(sqlStatement, idpAuthCredential, inviteCode)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error loading user from invite code %v: %w", inviteCode, err)
	}
	return true, nil
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
