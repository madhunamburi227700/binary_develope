CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- for gen_random_uuid() if preferred

-- ENUMs
CREATE TYPE verification_status_t AS ENUM ('FIXED','UNFIXED');
CREATE TYPE scan_status_t AS ENUM ('pending','fail','completed','scanning');
CREATE TYPE remediation_status_t AS ENUM ('STARTED','FIX_PENDING','FIX_GENERATED','PR_RAISED','COMPLETED');

CREATE TABLE IF NOT EXISTS hubs (
  id              varchar(64)    PRIMARY KEY,     -- SSD team id (bounded to save space)
  name            varchar(255)   NOT NULL UNIQUE,
  description     text,
  owner_email     varchar(254),                      -- owner email from SSD
  collaborators   text[],                            -- array of emails from SSD
  created_at      timestamptz   NOT NULL DEFAULT now(),
  updated_at      timestamptz   NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_hubs_owner_email ON hubs(owner_email);

CREATE TABLE IF NOT EXISTS projects (
  id              varchar(64)    PRIMARY KEY,
  name            varchar(255)   NOT NULL,
  hub_id          varchar(64)    NOT NULL,
  integration_id  varchar(255)   NOT NULL,
  organisation    varchar(255)   NOT NULL,
  last_scanned_time        timestamptz,                  -- last time project was scanned
  scheduled_time           integer,                      -- scan schedule duration in seconds
  created_at      timestamptz    NOT NULL DEFAULT now(),
  updated_at      timestamptz    NOT NULL DEFAULT now()
);


CREATE TABLE IF NOT EXISTS scans (
  id              varchar(64)    PRIMARY KEY,       -- SSD scan id
  parent_scan_id  varchar(64),                         -- parent SSD scan id
  project_id      varchar(64),                         -- SSD project id
  status          scan_status_t  NOT NULL DEFAULT 'pending',
  triggered_by    varchar(254),                         -- user email
  hub_id          varchar(64),                         -- SSD hub id
  remediated      integer         DEFAULT 0,           -- consolidated remediated count
  repository      varchar(512),
  branch          varchar(256),
  commit_sha      varchar(128),                         -- allow up to 128 to support sha256 if needed
  pull_request_id varchar(128),
  tag             varchar(128),
  settings        jsonb,                                -- runtime settings for this scan
  start_time      timestamptz,
  end_time        timestamptz,
  created_at      timestamptz     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_scans_project ON scans(project_id);
CREATE INDEX IF NOT EXISTS idx_scans_status ON scans(status);
CREATE INDEX IF NOT EXISTS idx_scans_triggered_by ON scans(triggered_by);
CREATE INDEX IF NOT EXISTS idx_scans_created_at ON scans(created_at);
CREATE INDEX IF NOT EXISTS idx_scans_start_time ON scans(start_time);
CREATE INDEX IF NOT EXISTS idx_scans_settings_gin ON scans USING GIN (settings);

-- Fast lookup by repository/branch/commit
CREATE INDEX IF NOT EXISTS idx_scans_repo_branch ON scans(repository, branch);

CREATE TABLE IF NOT EXISTS scan_type (
  id             varchar(80)    PRIMARY KEY,           -- e.g., "{scanid}-{type}" or SSD-provided id
  scan_id        varchar(64)    NOT NULL,              -- refers to scans.id 
  hub_id         varchar(64)    NOT NULL,
  scan_type      varchar(32)    NOT NULL,              -- sca, sast, etc.
  tool           varchar(128),
  file_name      varchar(512),
  file_url       varchar(1024),
  raw_json       jsonb,
  findings_count integer        DEFAULT 0,
  critical_count integer        DEFAULT 0,
  high_count     integer        DEFAULT 0,
  medium_count   integer        DEFAULT 0,
  low_count      integer        DEFAULT 0,
  unknown_count  integer        DEFAULT 0,
  CONSTRAINT fk_scan_type_scan FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_scan_type_scanid ON scan_type(scan_id);
CREATE INDEX IF NOT EXISTS idx_scan_type_tool ON scan_type(tool);
CREATE INDEX IF NOT EXISTS idx_scan_type_rawjson_gin ON scan_type USING GIN (raw_json);

CREATE TABLE IF NOT EXISTS vulnerabilities (
  id           uuid           PRIMARY KEY DEFAULT gen_random_uuid(),
  scan_id      varchar(64)    NOT NULL,                 -- refers to scans.id
  hub_id       varchar(64)    NOT NULL,
  name         varchar(800)   NOT NULL,                 -- rule/file/line or CVE id or hashed composite
  scan_type    varchar(32)    NOT NULL,                           -- 'sast' or 'sca' etc
  tool         varchar(128)   NOT NULL,
  package      varchar(128)   NOT NULL,
  version      varchar(128)   NOT NULL,
  metadata     jsonb          NOT NULL,
  severity     varchar(32)    NOT NULL,
  description  text         ,
  created_at   timestamptz    NOT NULL DEFAULT now(),
  CONSTRAINT fk_vuln_scan FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_vuln_scanid ON vulnerabilities(scan_id);
CREATE INDEX IF NOT EXISTS idx_vuln_severity ON vulnerabilities(severity);
CREATE INDEX IF NOT EXISTS idx_vuln_metadata_gin ON vulnerabilities USING GIN (metadata);
CREATE UNIQUE INDEX IF NOT EXISTS unique_idx_vuln_scanid_name_scan_type_tool_package_version
ON vulnerabilities (scan_id, name, scan_type, tool, package, version);

CREATE TABLE IF NOT EXISTS remediations (
  id               uuid                PRIMARY KEY,
  vulnerability_id uuid                NOT NULL,
  status           remediation_status_t NOT NULL,
  fix_commit_sha   varchar(128),
  fix_branch       varchar(256),
  pr_link          varchar(1024),
  prompt_id        uuid,
  conversation     TEXT[] DEFAULT ARRAY[]::TEXT[],                   -- AI-assisted conversation / comments
  created_at       timestamptz,
  updated_at       timestamptz,
  completed_at     timestamptz
);

CREATE INDEX IF NOT EXISTS idx_remediations_vuln ON remediations(vulnerability_id);
CREATE INDEX IF NOT EXISTS idx_remediations_status ON remediations(status);

CREATE TABLE IF NOT EXISTS audit_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id VARCHAR(255) NOT NULL,         -- ID of the authenticated user
  http_method VARCHAR(10),               -- HTTP method (e.g., GET, POST, etc.)
  action VARCHAR(100),                   -- Action (e.g. LOGIN, LOGOUT, SCAN_INITIATED)
  endpoint TEXT,                         -- The API endpoint (e.g., /api/v1/hubs)
  entity_name TEXT,                      -- Extracted entity name (e.g. 'projects', 'hubs')
  entity_id TEXT,                        -- Extracted entity ID (e.g. '123', '456')
  request_body TEXT,                     -- Request payload as string
  response_status SMALLINT,              -- HTTP response status code (e.g., 200, 404, 500)
  response_body TEXT,                    -- Response payload as string
  duration_ms INT,                       -- Time taken for the request (in ms)
  service_name VARCHAR(100),             -- Service name (for microservices)
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW() -- Timestamp of the log
);

CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255),
    name VARCHAR(255),
    provider VARCHAR(50) NOT NULL,           -- e.g. 'github', 'google', etc.
    provider_user_id VARCHAR(255) NOT NULL,  -- unique ID from provider
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT unique_idx_provider_user_id UNIQUE (provider_user_id)
);

CREATE TABLE user_sessions (
	id TEXT NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now(),
	last_accessed timestamptz,
	CONSTRAINT user_sessions_pkey PRIMARY KEY (id)
);

CREATE TABLE nli (
	id UUID NOT NULL,
	hub_id UUID NULL,
	status VARCHAR(100),
	conversation TEXT[],
	agents _text TEXT[],
	created_at TIMESTAMPTZ DEFAULT NOW(),
	updated_at TIMESTAMPTZ DEFAULT NOW(),
	CONSTRAINT nli_pkey PRIMARY KEY (id)
);
CREATE INDEX idx_nli_hub_id ON nli USING btree (hub_id);