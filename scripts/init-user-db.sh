#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE USER ep2 WITH PASSWORD 'ep2';
    CREATE DATABASE enterpriseportal2;
    GRANT CONNECT ON DATABASE enterpriseportal2 TO ep2;
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "enterpriseportal2" < /tmp/00schema.sql
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "enterpriseportal2" < /tmp/01seed.sql

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "enterpriseportal2" <<-EO1SQL
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ep2;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ep2;
EO1SQL
