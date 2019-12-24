package dao

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

const (
	ServiceAccountCreatePermission = "serviceaccount.create.execute"
	UserCreatePermission           = "user.create.execute"
	OrganizationCreatePermission   = "organization.create.execute"
)

type DaoHandler interface {
	Open() error
	Close() error
	TrySelect()
	GetNextUniqueId() int64
	CreateOrganization(*Organization) error
	AssignOrganizationToParent(parentId, orgID int64) (bool, error)
	CreateInviteForUser(organizationId int64, name string) (string, error)
	LoadUserFromInviteCode(inviteCode string) (*OrganizationUser, error)
	LoadUserFromCredential(credential string) (*OrganizationUser, error)
	InitUserFromInviteCode(inviteCode, idpAuthCredential string) (bool, error)
	LogUserIn(idpAuthCredential string) (*OrganizationUser, error)
	LoadOrganizationsForUser(userID int64) (map[int64]*Organization, error)
	LoadOrganizationDetails(organizationID int64) (*Organization, error)
	LoadServiceAccountCredentials(organizationId int64) (*ServiceAccountCredentials, error)
	CanUserViewOrg(userID, organizationID int64) (bool, error)

	DoesUserHavePermission(userID, organizationID int64, permission string) (bool, error)
}

type Dao struct {
	Db *sql.DB
}

func NewDaoHandler() DaoHandler {
	return &Dao{Db: nil}
}

func (d *Dao) DoesUserHavePermission(userID, organizationID int64, permission string) (bool, error) {
	// Test if any of the orgs between the root of the user and the org they are acting on (including
	// themselves contain the necessary role w/ permission.
	sqlStatement := `
		SELECT
				count(1)
		FROM
				organization_organization_user_role_xref 
		WHERE TRUE
				AND organization_id IN
				(SELECT id FROM organization WHERE path <@ 
				  (SELECT path FROM organization WHERE id IN 
					  (SELECT organization_id FROM organization_organization_user_xref 
						WHERE organization_user_id=$1)) AND path @> (SELECT path FROM organization WHERE id=$2))
				AND role_id IN 
				(SELECT r.id FROM role r, permission p, role_permission_xref rpx WHERE
				p.id = rpx.permission_id AND r.id = rpx.role_id AND p.value = $3)
`
	var count int
	row := d.Db.QueryRow(sqlStatement, userID, organizationID, permission)

	var err error
	err = row.Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("error checking permissions for user: %w", err)
	}

	return count > 0, nil
}

func (d *Dao) AssignOrganizationToParent(parentID int64, orgID int64) (bool, error) {
	sqlStatement := `
		UPDATE
			organization	
		SET
			path = (SELECT path FROM organization WHERE id = $1) || CAST($2 as TEXT)
		WHERE
			id = $2
`
	_, err := d.Db.Exec(sqlStatement, parentID, orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error adding organization to parent %w", err)
	}
	return true, nil
}

func (d *Dao) CanUserViewOrg(userID, organizationID int64) (bool, error) {
	sqlStatement := ` 
	SELECT
		count(1)
	FROM
		organization
	WHERE TRUE
		AND path <@ (SELECT path FROM organization WHERE id IN (SELECT organization_id FROM organization_organization_user_xref WHERE organization_user_id=$1))
		AND id=$2
	GROUP BY
		path
`
	var count int
	row := d.Db.QueryRow(sqlStatement, userID, organizationID)

	var err error
	err = row.Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("error loading orgs to see if user can view: %w", err)
	}

	return count > 0, nil
}
func (d *Dao) LoadServiceAccountCredentials(organizationId int64) (*ServiceAccountCredentials, error) {
	// find my first parent that has a valid service account (will always terminate at the root)
	sqlStatement := `
SELECT
    orgid, mat, mac
FROM
    (SELECT
         orgid,
         ordernum,
         (SELECT master_account_type FROM organization WHERE id = orgid::bigint) mat,
         (SELECT master_account_credential FROM organization WHERE id = orgid::bigint) mac
     FROM
         organization o,
         regexp_split_to_table(o.path::text, E'\\.') WITH ORDINALITY x(orgid,ordernum)
     WHERE TRUE
       AND o.id = $1) as orgs_with_accts
WHERE
    mat IS NOT NULL
ORDER BY ordernum DESC LIMIT 1;
	`
	var credentials ServiceAccountCredentials
	var err error
	row := d.Db.QueryRow(sqlStatement, organizationId)

	var jsonCredentials string
	err = row.Scan(&credentials.OwningOrganizationID, &credentials.Type, &jsonCredentials)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("error loading user from credential: %w", err)
	}

	credentials.RawCredentials = []byte(jsonCredentials)

	err = json.Unmarshal(credentials.RawCredentials, &credentials.Credentials)
	if err != nil {
		return nil, fmt.Errorf("error loading user from credential: %w", err)
	}

	return &credentials, nil
}

func (d *Dao) Open() error {
	var err error

	dbConnectionString := os.Getenv("PGSQL_CONNECTION_STRING")
	if len(dbConnectionString) == 0 {
		panic("PGSQL_CONNECTION_STRING undefined")
	}
	d.Db, err = sql.Open("postgres", dbConnectionString)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (d *Dao) GetNextUniqueId() int64 {
	return rand.Int63()
}

func (d *Dao) LoadOrganizationDetails(organizationID int64) (*Organization, error) {
	ret := &Organization{}
	{
		sqlStatement := `
	SELECT
		id, display_name
	FROM
		organization
	WHERE
		id = $1
	`
		row := d.Db.QueryRow(sqlStatement, organizationID)
		err := row.Scan(&ret.ID, &ret.DisplayName)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("error loading organization id %d error: %w", organizationID, err)
		}
	}

	{
		sqlStatement := `
	SELECT 
		id,display_name
	FROM 
		organization_user 
	WHERE 
		id IN (SELECT organization_user_id FROM organization_organization_user_xref WHERE organization_id = $1)
		AND current_state = 1
	ORDER BY 
		display_name
	`
		var err error
		rows, err := d.Db.Query(sqlStatement, organizationID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			u := &OrganizationUser{}
			err = rows.Scan(&u.ID, &u.DisplayName)
			if err != nil {
				return nil, err
			}
			ret.Users = append(ret.Users, u)
		}
	}

	return ret, nil
}

func (d *Dao) LoadOrganizationsForUser(userID int64) (map[int64]*Organization, error) {
	sqlStatement := `
	SELECT 
		id,display_name,path
	FROM 
		organization 
	WHERE 
		path <@ (SELECT path FROM organization WHERE id IN (SELECT organization_id FROM organization_organization_user_xref WHERE organization_user_id = $1))
	ORDER BY 
		path
	`
	var err error
	rows, err := d.Db.Query(sqlStatement, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userOrgs := make(map[int64]*Organization)
	for rows.Next() {
		org := &Organization{}
		err = rows.Scan(&org.ID, &org.DisplayName, &org.Path)
		if err != nil {
			return nil, err
		}
		userOrgs[org.ID] = org
	}

	return userOrgs, nil
}

func (d *Dao) LogUserIn(idpAuthCredential string) (*OrganizationUser, error) {
	sqlStatement := `SELECT id, display_name, ARRAY(SELECT organization_id FROM organization_organization_user_xref WHERE organization_user_id = id) AS organizations FROM organization_user WHERE idp_type = 'AUTH0' AND idp_credential_value=$1 AND current_state=1`
	var orgUser OrganizationUser

	row := d.Db.QueryRow(sqlStatement, idpAuthCredential)
	err := row.Scan(&orgUser.ID, &orgUser.DisplayName, pq.Array(&orgUser.Organizations))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error loading user from credential: %w", err)
	}

	return &orgUser, nil
}

func (d *Dao) LoadUserFromCredential(credential string) (*OrganizationUser, error) {
	sqlStatement := `SELECT id, display_name, ARRAY(SELECT organization_id FROM organization_organization_user_xref WHERE organization_user_id = id), (current_state = 1) FROM organization_user WHERE idp_credential_value=$1`
	var orgUser OrganizationUser

	row := d.Db.QueryRow(sqlStatement, credential)
	err := row.Scan(&orgUser.ID, &orgUser.DisplayName, pq.Array(&orgUser.Organizations), &orgUser.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error loading user from credential %v: %w", credential, err)
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
	var err error
	inviteCode := fmt.Sprintf("%d", d.GetNextUniqueId())
	orgUserID := fmt.Sprintf("%d", d.GetNextUniqueId())

	sqlStatement := `
		INSERT INTO organization_user (id, display_name, invite_code, created_timestamp, current_state)
		VALUES ($1, $2, $3, $4, $5);
	`
	_, err = d.Db.Exec(sqlStatement, orgUserID, name, inviteCode, "NOW()", 0)
	if err != nil {
		panic(err)
	}

	sqlRefStatement := `
		INSERT INTO organization_organization_user_xref (organization_id, organization_user_id) VALUES ($1, $2);
	`
	_, err = d.Db.Exec(sqlRefStatement, organizationId, orgUserID)
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
