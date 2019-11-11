CREATE TABLE
    organization
(
    id BIGINT PRIMARY KEY,
    display_name TEXT,
    master_account_type TEXT,
    master_account_credential TEXT,
    created_timestamp TIMESTAMP,
    current_state INT,
    path ltree
);

CREATE INDEX path_gist_idx ON organization USING gist(path);
CREATE INDEX path_idx ON organization USING btree(path);

/*
User: map[aud:***REMOVED*** exp:1.573055384e+09 family_name:Massey given_name:David iat:1.573019384e+09 
iss:https://***REMOVED***.auth0.com/ locale:en name:David Massey nickname:dmassey 
picture:httpse//lh3.googleusercontent.com/a-/AAuE7mAO-D5x6lVa_jy2zUP7D6IFz9O7yOTiIHwC-O2_ 
sub:google-oauth2|111861164484074538139 updated_at:2019-11-06T05:49:43.694Z]
*/

CREATE TABLE
    organization_user
(
    id BIGINT PRIMARY KEY,
    display_name TEXT,
    idp_type TEXT,
    idp_credential_value TEXT UNIQUE,
    organizations BIGINT ARRAY,
    invite_code TEXT,
    current_state INT,

    last_login_timestamp TIMESTAMP
)