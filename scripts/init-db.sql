-- Initialize CirrusSync Database
-- This script is run when the PostgreSQL container starts for the first time

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Set timezone
SET timezone = 'UTC';

-- Create database user with appropriate permissions (if not exists)
DO
$do$
BEGIN
   IF NOT EXISTS (
      SELECT FROM pg_catalog.pg_roles
      WHERE  rolname = 'cirrussync') THEN

      CREATE ROLE cirrussync LOGIN PASSWORD 'cirrussync_password';
   END IF;
END
$do$;

-- Grant necessary permissions
GRANT ALL PRIVILEGES ON DATABASE cirrussync TO cirrussync;
GRANT ALL ON SCHEMA public TO cirrussync;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO cirrussync;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO cirrussync;

-- Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO cirrussync;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO cirrussync;

-- Create indexes that might be useful for performance
-- Note: GORM will create the main tables and indexes, but we can prepare some common ones

-- Function to generate secure random strings
CREATE OR REPLACE FUNCTION generate_random_string(length INTEGER)
RETURNS TEXT AS $$
BEGIN
    RETURN encode(gen_random_bytes(length), 'hex');
END;
$$ LANGUAGE plpgsql;

-- Function to generate UUIDs with prefix
CREATE OR REPLACE FUNCTION generate_prefixed_id(prefix TEXT)
RETURNS TEXT AS $$
BEGIN
    RETURN prefix || '_' || replace(gen_random_uuid()::text, '-', '');
END;
$$ LANGUAGE plpgsql;

-- Initial configuration data (if needed)
-- This could include default plans, system settings, etc.
-- Tables will be created by GORM migrations, so we don't create them here

-- Log the initialization
INSERT INTO pg_stat_statements_reset();

-- Comment with initialization info
COMMENT ON DATABASE cirrussync IS 'CirrusSync API Database - Initialized on ' || now()::text;
