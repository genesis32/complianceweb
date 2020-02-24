package dao

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/genesis32/complianceweb/utils"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

const (
	UserReadExecutePermissionFlag = 1
)

type RegisteredResourcesStore map[string]*RegisteredResource
type SettingsStore map[string]*Setting
type UserRoleStore map[int64][]Role

type AuditMetadata map[string]interface{}

func (a AuditMetadata) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Make the Attrs struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields.
func (a *AuditMetadata) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &a)
}

type AuditRecord struct {
	ID                 int64
	CreatedTimestamp   time.Time
	OrganizationUserID int64
	OrganizationID     int64
	InternalKey        string
	Method             string
	Metadata           AuditMetadata
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

type DaoHandler interface {
	Open()
	Close() error
	TrySelect()

	LoadMetadataInTree(organizationId int64, key string) (int64, []byte)
	LoadOrganizationMetadata(organizationID int64) OrganizationMetadata
	UpdateOrganizationMetadata(organizationID int64, metadata OrganizationMetadata)

	CreateOrganization(*Organization)
	AssignOrganizationToParent(parentId, orgID int64) bool
	LoadOrganizationsForUser(userID int64) map[int64]*Organization
	LoadOrganizationDetails(organizationID int64, permissionFlags uint) *Organization

	CreateInviteForUser(organizationId int64, name string) (int64, int64)

	LoadUserFromInviteCode(inviteCode int64) *OrganizationUser
	LoadUserFromCredential(credential string, state int) *OrganizationUser
	LoadUserFromID(id int64) *OrganizationUser
	UpdateUserState(id int64, state int)

	InitUserFromInviteCode(inviteCode, idpAuthCredential string) bool
	LogUserIn(idpAuthCredential string) (*OrganizationUser, error)
	CanUserViewOrg(userID, organizationID int64) bool

	DoesUserHavePermission(userID, organizationID int64, permission string) bool
	DoesUserHaveSystemPermission(userID int64, permission string) bool

	UpdateSettings(settings ...*Setting) error
	GetSettings(key ...string) SettingsStore

	SetRolesToUser(userID, organizationID int64, roleNames []string)
	LoadEnabledResources() RegisteredResourcesStore

	HasValidRoles(roles []string) bool

	CreateAuditRecord(record *AuditRecord)
	SealAuditRecord(record *AuditRecord)
}

type Dao struct {
	Db *sql.DB
}

func (d *Dao) CreateAuditRecord(record *AuditRecord) {
	sqlStatement := `
		INSERT INTO
			resource_audit_log
		(id, created, current_state, organization_user_id, organization_id, internal_key, method)
		VALUES
		($1, $2, 0, $3, $4, $5, $6)
`
	_, err := d.Db.Exec(sqlStatement, record.ID, record.CreatedTimestamp, record.OrganizationUserID, record.OrganizationID, record.InternalKey, record.Method)
	if err != nil {
		log.Fatal(err)
	}

}

func (d *Dao) SealAuditRecord(record *AuditRecord) {
	sqlStatement := `
		UPDATE 
			resource_audit_log
		SET
		human_readable = $1,
		metadata = $3,
		current_state = 1
		WHERE
			id = $2 AND
			current_state = 0
`
	_, err := d.Db.Exec(sqlStatement, record.HumanReadable, record.ID, record.Metadata)
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Dao) HasValidRoles(roles []string) bool {

	sqlStatement := `
		SELECT 
			COUNT(1)
		FROM
			role
		WHERE
			display_name = ANY($1)
`
	var cnt int
	row := d.Db.QueryRow(sqlStatement, pq.Array(roles))
	err := row.Scan(&cnt)
	if err != nil {
		log.Fatal(err)
	}
	return cnt == len(roles)
}

func (d *Dao) UpdateUserState(id int64, state int) {
	sqlStatement := `
		UPDATE organization_user SET current_state = $2 WHERE id = $1 
`
	_, err := d.Db.Exec(sqlStatement, id, state)

	if err != nil {
		log.Fatalf("error updating state of user %d to %d err: %v", id, state, err)
	}
}

func (d *Dao) LoadUserFromID(id int64) *OrganizationUser {
	var ret OrganizationUser
	{
		sqlStatement := `
			SELECT
				id, display_name, current_state
			FROM
				organization_user 
			WHERE
				id = $1
`
		row := d.Db.QueryRow(sqlStatement, id)
		err := row.Scan(&ret.ID, &ret.DisplayName, &ret.CurrentState)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	ret.UserRoles = make(UserRoleStore)
	{
		sqlStatement := `
		SELECT 
			organization_id, role_id, (SELECT display_name FROM role where id = role_id) 
		FROM 
			organization_organization_user_role_xref 
		WHERE 
			organization_user_id = $1
`
		rows, err := d.Db.Query(sqlStatement, id)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		for rows.Next() {
			var roleID int64
			var roleName string
			var organizationID sql.NullInt64
			err = rows.Scan(&organizationID, &roleID, &roleName)
			if err != nil {
				log.Fatal(err)
			}
			if organizationID.Valid {
				ret.UserRoles[organizationID.Int64] = append(ret.UserRoles[organizationID.Int64], Role{ID: roleID, DisplayName: roleName})
				ret.Organizations = append(ret.Organizations, organizationID.Int64)
			}
		}
	}
	return &ret
}

func (d *Dao) UpdateOrganizationMetadata(organizationID int64, metadata OrganizationMetadata) {
	sqlStatement := `
		UPDATE organization SET metadata = $2 WHERE id = $1 
`
	_, err := d.Db.Exec(sqlStatement, organizationID, metadata)

	if err != nil {
		log.Fatalf("error updating metadata %w", err)
	}
}

func (d *Dao) LoadOrganizationMetadata(organizationID int64) OrganizationMetadata {
	sqlStatement := `SELECT metadata FROM organization WHERE id = $1`
	var ret OrganizationMetadata

	row := d.Db.QueryRow(sqlStatement, organizationID)
	err := row.Scan(&ret)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		log.Fatal(err)
	}

	return ret
}

func NewDaoHandler(db *sql.DB) *Dao {
	return &Dao{Db: db}
}

func (d *Dao) SetRolesToUser(organizationID, userID int64, roleNames []string) {
	tx, _ := d.Db.Begin()

	sqlStatement := `
		DELETE FROM
			organization_organization_user_role_xref
		WHERE 
			organization_id = $1 
			AND organization_user_id = $2
`
	_, err := tx.Exec(sqlStatement, organizationID, userID)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	for i := range roleNames {
		sqlStatement := `
		INSERT INTO
				organization_organization_user_role_xref
		(organization_id, organization_user_id, role_id)
		VALUES
				(NULLIF($1::bigint,0), $2, (SELECT id FROM role WHERE display_name = $3))
`
		_, err := tx.Exec(sqlStatement, organizationID, userID, roleNames[i])
		if err != nil {
			tx.Rollback()
			log.Fatal(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Dao) UpdateSettings(settings ...*Setting) error {

	tx, _ := d.Db.Begin()

	for _, s := range settings {

		sqlStatement := `
		INSERT INTO
				settings
		(key, value)
		VALUES
		($1, $2)
		ON CONFLICT (key) DO
		UPDATE
		SET value = $2
`
		_, err := tx.Exec(sqlStatement, s.Key, s.Value)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error updating settings %w", err)
		}
	}
	tx.Commit()

	return nil
}

func (d *Dao) GetSettings(keys ...string) SettingsStore {

	sqlStatement := `
		SELECT
				key, value
		FROM
				settings
		WHERE
				key = ANY($1)
`
	rows, err := d.Db.Query(sqlStatement, pq.Array(keys))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	ret := make(SettingsStore)
	for rows.Next() {
		s := &Setting{}
		err = rows.Scan(&s.Key, &s.Value)
		if err != nil {
			log.Fatal(err)
		}
		ret[s.Key] = s
	}

	return ret
}

func (d *Dao) DoesUserHaveSystemPermission(userID int64, permission string) bool {
	// TODO: Verify that this has permission starts with system.
	sqlStatement := `
				SELECT
						count(1)
				FROM
					organization_organization_user_role_xref 
				WHERE 
						organization_id IS NULL AND 
						organization_user_id = $1 AND 
						role_id IN 
						(SELECT r.id FROM role r, permission p, role_permission_xref rpx WHERE
				p.id = rpx.permission_id AND r.id = rpx.role_id AND p.value = $2)
`
	var count int
	row := d.Db.QueryRow(sqlStatement, userID, permission)

	var err error
	err = row.Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}

	if err != nil {
		log.Fatal(err)
	}

	return count > 0
}

func (d *Dao) DoesUserHavePermission(userID, organizationID int64, permission string) bool {
	// Test if any of the orgs between the root of the user and the org they are acting on (including
	// themselves contain the necessary role w/ permission.
	sqlStatement := `
		SELECT
				count(1)
		FROM
				organization_organization_user_role_xref 
		WHERE 
				(organization_id IN (SELECT id FROM organization WHERE path @> (SELECT path FROM organization WHERE id=$2)) AND
				role_id IN (SELECT r.id FROM role r, permission p, role_permission_xref rpx WHERE p.id = rpx.permission_id AND r.id = rpx.role_id AND p.value = $3)) AND
				organization_user_id = $1
`
	var count int
	row := d.Db.QueryRow(sqlStatement, userID, organizationID, permission)

	var err error
	err = row.Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}

	if err != nil {
		log.Fatal(err)
	}

	return count > 0
}

func (d *Dao) AssignOrganizationToParent(parentID int64, orgID int64) bool {
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
		return false
	}

	if err != nil {
		log.Fatalf("error adding organization to parent %w", err)
	}

	return true
}

func (d *Dao) CanUserViewOrg(userID, organizationID int64) bool {
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
		return false
	}

	if err != nil {
		log.Fatal(err)
	}

	return count > 0
}
func (d *Dao) LoadMetadataInTree(organizationId int64, key string) (int64, []byte) {
	// find my first parent that has a valid service account (will always terminate at the root)
	sqlStatement := `
SELECT
    orgid, metadata
FROM
    (SELECT
         orgid,
         ordernum,
         (SELECT metadata FROM organization WHERE id = orgid::bigint) metadata
     FROM
         organization o,
         regexp_split_to_table(o.path::text, E'\\.') WITH ORDINALITY x(orgid,ordernum)
     WHERE TRUE
       AND o.id = $1) as orgs_with_metadata
WHERE
    metadata->>$2 IS NOT NULL
ORDER BY ordernum DESC LIMIT 1;
	`
	row := d.Db.QueryRow(sqlStatement, organizationId, key)

	var organizationID int64
	var organizationMetadata []byte
	err := row.Scan(&organizationID, &organizationMetadata)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, []byte{}
	}

	if err != nil {
		log.Fatal(err)
	}

	return organizationID, organizationMetadata
}

func (d *Dao) Open() {
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

func (d *Dao) LoadOrganizationDetails(organizationID int64, permissionFlags uint) *Organization {
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
			return nil
		}
		if err != nil {
			log.Fatalf("error loading organization id %d error: %w", organizationID, err)
		}
	}

	if (permissionFlags & UserReadExecutePermissionFlag) == UserReadExecutePermissionFlag {
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
			log.Fatal(err)
		}
		defer rows.Close()

		ret.Users = make([]*OrganizationUser, 0)
		for rows.Next() {
			u := &OrganizationUser{}
			err = rows.Scan(&u.ID, &u.DisplayName)
			if err != nil {
				log.Fatal(err)
			}
			ret.Users = append(ret.Users, u)
		}
	}
	return ret
}

func (d *Dao) LoadOrganizationsForUser(userID int64) map[int64]*Organization {
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
		log.Fatal(err)
	}
	defer rows.Close()

	userOrgs := make(map[int64]*Organization)
	for rows.Next() {
		org := &Organization{}
		err = rows.Scan(&org.ID, &org.DisplayName, &org.Path)
		if err != nil {
			log.Fatal(err)
		}
		userOrgs[org.ID] = org
	}

	return userOrgs
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

const (
	UserCreatedState  = 0
	UserActiveState   = 1
	UserDeactiveState = 2
)

func (d *Dao) LoadUserFromCredential(credential string, state int) *OrganizationUser {
	sqlStatement := `SELECT id, display_name, ARRAY(SELECT organization_id FROM organization_organization_user_xref WHERE organization_user_id = id), current_state FROM organization_user WHERE idp_credential_value=$1 AND current_state=$2`
	var orgUser OrganizationUser

	row := d.Db.QueryRow(sqlStatement, credential, state)
	err := row.Scan(&orgUser.ID, &orgUser.DisplayName, pq.Array(&orgUser.Organizations), &orgUser.CurrentState)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		log.Fatalf("error loading user from credential %v: %w", credential, err)
	}

	return &orgUser
}
func (d *Dao) LoadUserFromInviteCode(inviteCode int64) *OrganizationUser {
	sqlStatement := `SELECT id, display_name FROM organization_user WHERE invite_code=$1 AND current_state=0`
	var orgUser OrganizationUser

	row := d.Db.QueryRow(sqlStatement, inviteCode)
	err := row.Scan(&orgUser.ID, &orgUser.DisplayName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		log.Fatalf("error loading user from invite code %v: %w", inviteCode, err)
	}

	return &orgUser
}

func (d *Dao) CreateInviteForUser(organizationId int64, name string) (int64, int64) {
	var err error
	orgUserID := utils.GetNextUniqueId()
	inviteCode := utils.GetNextUniqueId()

	sqlStatement := `
		INSERT INTO organization_user (id, display_name, invite_code, created_timestamp, current_state)
		VALUES ($1, $2, $3, $4, $5);
	`
	_, err = d.Db.Exec(sqlStatement, orgUserID, name, inviteCode, "NOW()", 0)
	if err != nil {
		log.Fatal(err)
	}

	if organizationId != 0 {
		sqlRefStatement := `
INSERT INTO organization_organization_user_xref (organization_id, organization_user_id) VALUES ($1, $2);
	`
		_, err = d.Db.Exec(sqlRefStatement, organizationId, orgUserID)
		if err != nil {
			log.Fatal(err)
		}
	}

	return orgUserID, inviteCode
}

func (d *Dao) CreateOrganization(org *Organization) {
	sqlStatement := `
	INSERT INTO organization (id, display_name, metadata, path)
	VALUES ($1, $2, '{}', $3)
	`
	_, err := d.Db.Exec(sqlStatement, org.ID, org.DisplayName, fmt.Sprintf("%d", org.ID))
	if err != nil {
		log.Fatal(err)
	}
}

func (d *Dao) InitUserFromInviteCode(inviteCode, idpAuthCredential string) bool {
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
		return false
	}
	if err != nil {
		log.Fatal("error loading user from invite code %v: %w", inviteCode, err)
	}
	return true
}

func (d *Dao) LoadEnabledResources() RegisteredResourcesStore {
	sqlStatement := `
		SELECT
				id, display_name, internal_key
		FROM
				registered_resources
		WHERE
				enabled = true
`
	rows, err := d.Db.Query(sqlStatement)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	ret := make(RegisteredResourcesStore)
	for rows.Next() {
		s := &RegisteredResource{Enabled: true}
		err = rows.Scan(&s.ID, &s.DisplayName, &s.InternalKey)
		if err != nil {
			log.Fatal(err)
		}
		ret[s.InternalKey] = s
	}

	return ret
}

func (d *Dao) TrySelect() {
	sqlStatement := `SELECT id FROM organization WHERE display_name='baz'`
	row := d.Db.QueryRow(sqlStatement)
	var out int
	err := row.Scan(&out)
	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}
}

func (d *Dao) Close() error {
	err := d.Db.Close()
	return err
}
