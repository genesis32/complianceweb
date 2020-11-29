create extension ltree;

CREATE TABLE IF NOT EXISTS
organization
(
  id BIGINT PRIMARY KEY,
  display_name TEXT,
  created_timestamp TIMESTAMP,
  current_state INT,
  metadata jsonb,
  path ltree
);

CREATE INDEX IF NOT EXISTS path_gist_idx ON organization USING gist(path);
CREATE INDEX IF NOT EXISTS path_idx ON organization USING btree(path);

CREATE TABLE IF NOT EXISTS
organization_organization_user_xref
( 
  organization_id BIGINT,
  organization_user_id BIGINT
);

CREATE TABLE IF NOT EXISTS 
organization_user
(
  id BIGINT PRIMARY KEY,
  display_name TEXT,
  idp_type TEXT,
  idp_credential_value TEXT UNIQUE,
  invite_code TEXT,
  current_state INT,
  last_login_timestamp TIMESTAMP,
  created_timestamp TIMESTAMP
);

CREATE TABLE IF NOT EXISTS
organization_organization_user_role_xref (
    organization_id BIGINT,
    organization_user_id BIGINT,
    role_id BIGINT
);

CREATE TABLE IF NOT EXISTS
role_permission_xref (
    role_id BIGINT,
    permission_id BIGINT
);

CREATE TABLE IF NOT EXISTS
role
(
   id BIGINT PRIMARY KEY,
   display_name TEXT
);

CREATE TABLE IF NOT EXISTS
permission
(
    id BIGINT PRIMARY KEY,
    display_name TEXT,
    value TEXT 
);

CREATE TABLE IF NOT EXISTS
organization_organization_user_role_xref (
    organization_id BIGINT,
    organization_user_id BIGINT,
    role_id BIGINT
);

CREATE TABLE IF NOT EXISTS
settings (
    key TEXT PRIMARY KEY,
    value TEXT
);

CREATE TABLE IF NOT EXISTS
registered_resources (
    id BIGINT PRIMARY KEY,
    display_name TEXT,
    internal_key TEXT,
    enabled BOOLEAN
);

CREATE TABLE resource_audit_log (
    id BIGINT PRIMARY KEY,
    created TIMESTAMP,
    current_state INT,
    organization_user_id BIGINT,
    organization_id BIGINT,
    internal_key TEXT,
    method TEXT,
    metadata jsonb,
    human_readable TEXT
);
