-- Create the user
CREATE USER temporal WITH PASSWORD 'temporal';
ALTER USER temporal WITH SUPERUSER;

-- Create the 'temporal' database and assign ownership
CREATE DATABASE temporal OWNER temporal;

-- Create the 'temporal_visibility' database and assign ownership
CREATE DATABASE temporal_visibility OWNER temporal;

-- Connect to each database and grant schema permissions
\connect temporal
GRANT ALL PRIVILEGES ON SCHEMA public TO temporal;

\connect temporal_visibility
GRANT ALL PRIVILEGES ON SCHEMA public TO temporal;
