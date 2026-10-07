-- Setup External Secrets Operator PostgreSQL backend
-- Run this as PostgreSQL superuser (e.g. postgres)

-- Create dedicated user for External Secrets Operator
CREATE USER external_secrets WITH PASSWORD 'REPLACE_WITH_REAL_PASSWORD';

-- Grant schema usage
GRANT USAGE ON SCHEMA public TO external_secrets;

-- Grant permissions on external_secrets table
GRANT SELECT, INSERT, UPDATE, DELETE ON external_secrets TO external_secrets;
GRANT USAGE, SELECT ON SEQUENCE external_secrets_id_seq TO external_secrets;

-- Revoke default PUBLIC grants (security best practice)
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON DATABASE fitness_platform FROM PUBLIC;
