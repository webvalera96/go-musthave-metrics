-- migrations/000001_create_metrics_table.down.sql
DROP INDEX IF EXISTS idx_metrics_id;
DROP TABLE IF EXISTS metrics;