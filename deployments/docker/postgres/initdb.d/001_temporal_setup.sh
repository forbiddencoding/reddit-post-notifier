#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "postgres" <<-EOSQL
    CREATE USER temporal WITH PASSWORD 'temporal' SUPERUSER;
    CREATE DATABASE temporal OWNER temporal;
    CREATE DATABASE temporal_visibility OWNER temporal;
EOSQL

for db in temporal temporal_visibility; do
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$db" <<-EOSQL
    GRANT ALL PRIVILEGES ON SCHEMA public TO temporal;
EOSQL
done