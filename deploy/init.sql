-- Initialize the PostgreSQL database for GoTrack
-- This script is automatically executed when the postgres container starts

-- Create the events table with proper indexes
CREATE TABLE IF NOT EXISTS events_json (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID UNIQUE NOT NULL,
    ts TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload JSONB NOT NULL
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_events_json_ts ON events_json (ts);
CREATE INDEX IF NOT EXISTS idx_events_json_gin ON events_json USING GIN (payload);

-- Create additional indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_events_json_event_type ON events_json USING GIN ((payload->>'type'));
CREATE INDEX IF NOT EXISTS idx_events_json_visitor_id ON events_json USING GIN ((payload->'session'->>'visitor_id'));

-- Create a view for easier querying
CREATE OR REPLACE VIEW events_view AS
SELECT 
    id,
    event_id,
    ts,
    payload->>'type' as event_type,
    payload->'session'->>'visitor_id' as visitor_id,
    payload->'session'->>'session_id' as session_id,
    payload->'url'->>'referrer' as referrer,
    payload->'device'->>'browser' as browser,
    payload->'device'->>'os' as os,
    payload
FROM events_json;

-- Example queries for analytics (commented out)
/*
-- Count events by type
SELECT 
    payload->>'type' as event_type, 
    COUNT(*) as count 
FROM events_json 
WHERE ts >= NOW() - INTERVAL '1 day' 
GROUP BY payload->>'type';

-- Top referrers
SELECT 
    payload->'url'->>'referrer' as referrer, 
    COUNT(*) as visits 
FROM events_json 
WHERE payload->>'type' = 'pageview' 
    AND ts >= NOW() - INTERVAL '1 day'
    AND payload->'url'->>'referrer' IS NOT NULL 
GROUP BY payload->'url'->>'referrer' 
ORDER BY visits DESC 
LIMIT 10;

-- Browser distribution
SELECT 
    payload->'device'->>'browser' as browser, 
    COUNT(*) as count 
FROM events_json 
WHERE ts >= NOW() - INTERVAL '1 day' 
GROUP BY payload->'device'->>'browser' 
ORDER BY count DESC;
*/

-- Grant permissions (optional, adjust as needed)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO analytics;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO analytics;