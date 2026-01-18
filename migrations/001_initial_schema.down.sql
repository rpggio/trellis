-- Drop triggers first
DROP TRIGGER IF EXISTS records_au;
DROP TRIGGER IF EXISTS records_ad;
DROP TRIGGER IF EXISTS records_ai;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS records_fts;
DROP TABLE IF EXISTS activity_log;
DROP TABLE IF EXISTS session_activations;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS record_relations;
DROP TABLE IF EXISTS records;
DROP TABLE IF EXISTS projects;
