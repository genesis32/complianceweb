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

/*
User: map[aud:***REMOVED*** exp:1.573055384e+09 family_name:Massey given_name:David iat:1.573019384e+09 
iss:https://***REMOVED***.auth0.com/ locale:en name:David Massey nickname:dmassey 
picture:httpse//lh3.googleusercontent.com/a-/AAuE7mAO-D5x6lVa_jy2zUP7D6IFz9O7yOTiIHwC-O2_ 
sub:google-oauth2|111861164484074538139 updated_at:2019-11-06T05:49:43.694Z]
 */

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