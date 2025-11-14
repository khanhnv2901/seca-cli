-- PostgreSQL initialization script for SECA-CLI
-- This script runs automatically when the database is first created

-- Create schema version table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial version
INSERT INTO schema_migrations (version) VALUES ('init') ON CONFLICT DO NOTHING;

-- Create organizations table (for future multi-tenancy)
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    max_users INT DEFAULT 5,
    max_checks_per_month INT DEFAULT 1000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create users table (for future authentication)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'operator',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create engagements table
CREATE TABLE IF NOT EXISTS engagements (
    id BIGINT PRIMARY KEY,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    owner VARCHAR(255) NOT NULL,
    roe TEXT NOT NULL,
    roe_agree BOOLEAN NOT NULL DEFAULT false,
    scope TEXT[] NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create check_results table
CREATE TABLE IF NOT EXISTS check_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    engagement_id BIGINT NOT NULL REFERENCES engagements(id) ON DELETE CASCADE,
    target VARCHAR(2048) NOT NULL,
    check_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    http_status INT,
    checked_at TIMESTAMPTZ NOT NULL,
    response_time_ms FLOAT,
    security_headers JSONB,
    tls_compliance JSONB,
    cookie_findings JSONB,
    cors_insights JSONB,
    cache_policy JSONB,
    network_security JSONB,
    client_security JSONB,
    notes TEXT,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    engagement_id BIGINT REFERENCES engagements(id) ON DELETE CASCADE,
    operator VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    target VARCHAR(2048),
    timestamp TIMESTAMPTZ NOT NULL,
    duration_seconds FLOAT,
    status VARCHAR(50),
    sha256_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_engagements_org ON engagements(organization_id);
CREATE INDEX IF NOT EXISTS idx_engagements_status ON engagements(status);
CREATE INDEX IF NOT EXISTS idx_check_results_engagement ON check_results(engagement_id);
CREATE INDEX IF NOT EXISTS idx_check_results_checked_at ON check_results(checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_engagement ON audit_logs(engagement_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_operator ON audit_logs(operator);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add updated_at triggers
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_engagements_updated_at BEFORE UPDATE ON engagements
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample data for development
INSERT INTO organizations (id, name, plan, max_users, max_checks_per_month)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'Development Org', 'enterprise', 100, 100000)
ON CONFLICT DO NOTHING;

INSERT INTO users (id, organization_id, email, username, role)
VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'admin@seca.local', 'admin', 'admin'),
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'operator@seca.local', 'operator', 'operator')
ON CONFLICT DO NOTHING;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO seca;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO seca;

-- Log completion
INSERT INTO schema_migrations (version) VALUES ('initial_schema_v1') ON CONFLICT DO NOTHING;
