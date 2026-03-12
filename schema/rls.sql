/****************************************************************************************
 AI SAFE POSTGRES SETUP
 Role: nli_agent
 Purpose: allow Claude MCP to query limited tables safely with tenant isolation

 Tenant context injected via connection string:
 ?options=-c%20app.hub_id=<HUB_ID>
****************************************************************************************/

-- PHASE 1 — CREATE AI ROLE
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'nli_agent') THEN
        CREATE ROLE nli_agent LOGIN PASSWORD 'Network@1234';
    END IF;
END
$$;

-- Lock down privileges
ALTER ROLE nli_agent
NOSUPERUSER
NOCREATEDB
NOCREATEROLE
NOREPLICATION;

-- PHASE 2 — HELPER FUNCTION (Tenant context)

CREATE OR REPLACE FUNCTION get_current_hub()
RETURNS text
LANGUAGE sql
STABLE
AS $$
SELECT current_setting('app.hub_id', true)
$$;

-- PHASE 3 — ENABLE RLS
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE scans ENABLE ROW LEVEL SECURITY;
ALTER TABLE scan_type ENABLE ROW LEVEL SECURITY;
ALTER TABLE vulnerabilities ENABLE ROW LEVEL SECURITY;
ALTER TABLE remediations ENABLE ROW LEVEL SECURITY;

ALTER TABLE projects FORCE ROW LEVEL SECURITY;
ALTER TABLE scans FORCE ROW LEVEL SECURITY;
ALTER TABLE scan_type FORCE ROW LEVEL SECURITY;
ALTER TABLE vulnerabilities FORCE ROW LEVEL SECURITY;
ALTER TABLE remediations FORCE ROW LEVEL SECURITY;

-- PHASE 4 — RLS POLICIES
DROP POLICY IF EXISTS hub_select_policy ON projects;
CREATE POLICY hub_select_policy
ON projects
FOR SELECT
USING (
    get_current_hub() IS NOT NULL
    AND hub_id = get_current_hub()
);

DROP POLICY IF EXISTS hub_select_policy ON scans;
CREATE POLICY hub_select_policy
ON scans
FOR SELECT
USING (
    get_current_hub() IS NOT NULL
    AND hub_id = get_current_hub()
);

DROP POLICY IF EXISTS hub_select_policy ON scan_type;
CREATE POLICY hub_select_policy
ON scan_type
FOR SELECT
USING (
    get_current_hub() IS NOT NULL
    AND hub_id = get_current_hub()
);

DROP POLICY IF EXISTS hub_select_policy ON vulnerabilities;
CREATE POLICY hub_select_policy
ON vulnerabilities
FOR SELECT
USING (
    get_current_hub() IS NOT NULL
    AND hub_id = get_current_hub()
);

DROP POLICY IF EXISTS hub_select_policy ON remediations;
CREATE POLICY hub_select_policy
ON remediations
FOR SELECT
USING (
    get_current_hub() IS NOT NULL
    AND hub_id = get_current_hub()
);



-- PHASE 5 — INDEXES (critical for RLS performance)
CREATE INDEX IF NOT EXISTS idx_projects_hub_id ON projects(hub_id);
CREATE INDEX IF NOT EXISTS idx_scans_hub_id ON scans(hub_id);
CREATE INDEX IF NOT EXISTS idx_scan_type_hub_id ON scan_type(hub_id);
CREATE INDEX IF NOT EXISTS idx_vulnerabilities_hub_id ON vulnerabilities(hub_id);
CREATE INDEX IF NOT EXISTS idx_remediations_hub_id ON remediations(hub_id);



-- PHASE 6 — LOCK DOWN ACCESS

-- Remove all privileges
REVOKE ALL ON SCHEMA public FROM nli_agent;
REVOKE ALL ON ALL TABLES IN SCHEMA public FROM nli_agent;

-- Allow schema access
GRANT USAGE ON SCHEMA public TO nli_agent;


-- PHASE 7 — ALLOW ACCESS ONLY TO APPROVED TABLES

GRANT SELECT ON projects TO nli_agent;
GRANT SELECT ON scans TO nli_agent;
GRANT SELECT ON scan_type TO nli_agent;
GRANT SELECT ON vulnerabilities TO nli_agent;
GRANT SELECT ON remediations TO nli_agent;

-- PHASE 8 — BLOCK SCHEMA MODIFICATIONS

REVOKE CREATE ON SCHEMA public FROM nli_agent;

-- CONNECTION STRING FOR MCP

-- Claude MCP must connect like this:

-- postgres://nli_agent:password@host:5432/db?options=-c%20app.hub_id=<HUB_ID>

-- Example:
-- postgres://nli_agent:password@localhost:5432/postgres?options=-c%20app.hub_id=2171fb6f-f2ae-4c91-827d-94107c2fa224



-- SECURITY GUARANTEES

-- nli_agent can ONLY:
--   SELECT from 5 allowed tables

-- nli_agent CANNOT:
--   access other tables
--   write data
--   create objects
--   escalate privileges
--   bypass RLS

-- Even if AI runs:
--   SELECT * FROM vulnerabilities;

-- PostgreSQL enforces:
--   SELECT * FROM vulnerabilities
--   WHERE hub_id = get_current_hub();