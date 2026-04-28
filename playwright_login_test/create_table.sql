-- ClickHouse Table Schema for Playwright Login Tests
-- Database: vusmart
-- Table: monitoring.playwright_login

CREATE TABLE IF NOT EXISTS monitoring.playwright_login (
    timestamp DateTime64(3),
    test_id String,
    username String,
    success UInt8,
    total_response_time_ms Float64,
    api_response_time_ms Float64,
    ui_render_time_ms Float64,
    page_load_time_ms Float64,
    final_url String,
    error String DEFAULT ''
) ENGINE = MergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (test_id, timestamp)
SETTINGS index_granularity = 8192;
