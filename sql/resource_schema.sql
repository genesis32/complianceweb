CREATE TABLE resource_gcpserviceaccount (
    id BIGINT PRIMARY KEY,
    external_ref TEXT UNIQUE,
    state jsonb
)