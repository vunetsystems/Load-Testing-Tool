-- ClickHouse table for storing Playwright log analytics test results
CREATE TABLE IF NOT EXISTS monitoring.playwright_log_analytics (
    timestamp DateTime,
    username String,
    filter String,
    success UInt8,
    total_time_ms Float64,
    page_load_time_ms Float64,
    search_api_time_ms Float64,
    results_render_time_ms Float64,
    test_id String
) ENGINE = MergeTree()
ORDER BY (timestamp, username, filter);
