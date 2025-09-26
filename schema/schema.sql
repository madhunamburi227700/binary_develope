CREATE TABLE "users" (
  "id" uuid PRIMARY KEY,
  "email" varchar UNIQUE NOT NULL,
  "name" varchar,
  "status" varchar,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "hubs" (
  "id" uuid PRIMARY KEY,
  "name" varchar UNIQUE NOT NULL,
  "description" text,
  "owner_id" uuid,
  "collabrator_id" uuid [],
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "integrations" (
  "id" uuid PRIMARY KEY,
  "user_id" uuid NOT NULL,
  "type" varchar,
  "name" varchar,
  "config" text,
  "is_active" boolean,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "settings" (
  "id" uuid PRIMARY KEY,
  "hub_id" uuid,
  "key" varchar,
  "value" varchar,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "projects" (
  "id" uuid PRIMARY KEY,
  "hub_id" uuid,
  "integration_id" uuid,
  "name" varchar,
  "repo_url" varchar,
  "description" text,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "scans" (
  "id" uuid PRIMARY KEY,
  "project_id" uuid,
  "scan_type" varchar,
  "tool" varchar,
  "status" varchar,
  "triggered_by" uuid,
  "file_name" varchar,
  "file_url" varchar,
  "raw_json" jsonb,
  "findings_count" int,
  "critical_count" int,
  "high_count" int,
  "medium_count" int,
  "low_count" int,
  "remediated" int,
  "branch" varchar,
  "commit_sha" varchar,
  "pull_request_id" varchar,
  "tag" varchar,
  "settings" jsonb,
  "start_time" timestamp,
  "end_time" timestamp,
  "created_at" timestamp,
  "updated_at" timestamp
);

CREATE TABLE "vulnerabilities" (
  "id" uuid PRIMARY KEY,
  "scan_id" uuid NOT NULL,
  "name" varchar,
  "type" varchar,
  "metadata" jsonb,
  "severity" varchar,
  "description" varchar
);

CREATE TABLE "remediations" (
  "id" uuid PRIMARY KEY,
  "scan_result_id" uuid NOT NULL,
  "status" varchar,
  "fix_commit_sha" varchar,
  "pr_link" varchar,
  "prompt_id" uuid,
  "conversation" jsonb,
  "started_at" timestamp,
  "completed_at" timestamp
);

CREATE TABLE "remediation_verification" (
  "id" uuid PRIMARY KEY,
  "vulnerability_id" uuid,
  "remediation_id" uuid,
  "verification_tool" varchar,
  "status" varchar,
  "description" varchar,
  "created_at" timestamp
);

CREATE TABLE "remediation_feedback" (
  "id" uuid PRIMARY KEY,
  "remediation_id" uuid,
  "vulnerability_id" uuid,
  "comments" varchar,
  "rating" num
);

CREATE TABLE "audit_logs" (
  "id" uuid PRIMARY KEY,
  "user_id" uuid,
  "hub_id" uuid,
  "action" varchar,
  "metadata" jsonb,
  "created_at" timestamp
);

ALTER TABLE "hubs" ADD FOREIGN KEY ("owner_id") REFERENCES "users" ("id");

ALTER TABLE "integrations" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "settings" ADD FOREIGN KEY ("hub_id") REFERENCES "hubs" ("id");

ALTER TABLE "projects" ADD FOREIGN KEY ("hub_id") REFERENCES "hubs" ("id");

ALTER TABLE "projects" ADD FOREIGN KEY ("integration_id") REFERENCES "integrations" ("id");

ALTER TABLE "scans" ADD FOREIGN KEY ("project_id") REFERENCES "projects" ("id");

ALTER TABLE "scans" ADD FOREIGN KEY ("triggered_by") REFERENCES "users" ("id");

ALTER TABLE "vulnerabilities" ADD FOREIGN KEY ("scan_id") REFERENCES "scans" ("id");

ALTER TABLE "remediations" ADD FOREIGN KEY ("scan_result_id") REFERENCES "vulnerabilities" ("id");

ALTER TABLE "remediation_verification" ADD FOREIGN KEY ("vulnerability_id") REFERENCES "vulnerabilities" ("id");

ALTER TABLE "remediation_verification" ADD FOREIGN KEY ("remediation_id") REFERENCES "remediations" ("id");

ALTER TABLE "remediation_feedback" ADD FOREIGN KEY ("remediation_id") REFERENCES "remediations" ("id");

ALTER TABLE "remediation_feedback" ADD FOREIGN KEY ("vulnerability_id") REFERENCES "vulnerabilities" ("id");

ALTER TABLE "audit_logs" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");

ALTER TABLE "audit_logs" ADD FOREIGN KEY ("hub_id") REFERENCES "hubs" ("id");
