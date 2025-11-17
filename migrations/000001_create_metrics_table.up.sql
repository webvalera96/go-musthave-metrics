-- migrations/000001_create_metrics_table.up.sql
CREATE TABLE metrics (
    id VARCHAR(255) PRIMARY KEY,
    mtype VARCHAR(255) NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    hash VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Базовый индекс для поиска по id
CREATE INDEX idx_metrics_id ON metrics (id);