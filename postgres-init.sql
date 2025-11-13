-- PostgreSQL initialization script for HelixCode
-- Creates the required user and database

-- Create the helix user if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'helix') THEN
        CREATE USER helix WITH PASSWORD 'helixpass';
    END IF;
END
$$;

-- Create the database if it doesn't exist
SELECT 'CREATE DATABASE helixcode_prod'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'helixcode_prod')\gexec

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE helixcode_prod TO helix;

-- Connect to the database and create extensions
\c helixcode_prod

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Grant permissions to the helix user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO helix;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO helix;