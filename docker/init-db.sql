-- Initialize database schema for rate limiter system

-- Create rate limiting rules table
CREATE TABLE IF NOT EXISTS rate_limit_rules (
    id VARCHAR(255) PRIMARY KEY,
    resource VARCHAR(255) NOT NULL,
    limit_count INTEGER NOT NULL,
    window_duration INTERVAL NOT NULL,
    algorithm VARCHAR(50) NOT NULL DEFAULT 'sliding_window',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create rate limit events table for event sourcing
CREATE TABLE IF NOT EXISTS rate_limit_events (
    id VARCHAR(255) PRIMARY KEY,
    aggregate_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    version INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create read model for rate limit status
CREATE TABLE IF NOT EXISTS rate_limit_status (
    client_id VARCHAR(255) NOT NULL,
    resource VARCHAR(255) NOT NULL,
    request_count INTEGER DEFAULT 0,
    window_start TIMESTAMP,
    window_end TIMESTAMP,
    remaining_quota INTEGER DEFAULT 0,
    is_blocked BOOLEAN DEFAULT FALSE,
    blocked_until TIMESTAMP,
    last_request_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (client_id, resource)
);

-- Create rules table for rule engine
CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    rule_type VARCHAR(50) NOT NULL,
    description TEXT,
    priority INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT TRUE,
    conditions JSONB NOT NULL,
    actions JSONB NOT NULL,
    tags TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255)
);

-- Create rule evaluation history table
CREATE TABLE IF NOT EXISTS rule_evaluation_history (
    id SERIAL PRIMARY KEY,
    rule_id VARCHAR(255) REFERENCES rules(id),
    client_id VARCHAR(255),
    resource VARCHAR(255),
    matched BOOLEAN NOT NULL,
    evaluation_context JSONB,
    actions_taken JSONB,
    evaluated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create client statistics table
CREATE TABLE IF NOT EXISTS client_statistics (
    client_id VARCHAR(255) PRIMARY KEY,
    total_requests INTEGER DEFAULT 0,
    blocked_requests INTEGER DEFAULT 0,
    allowed_requests INTEGER DEFAULT 0,
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_rate_limit_events_aggregate_id ON rate_limit_events(aggregate_id);
CREATE INDEX IF NOT EXISTS idx_rate_limit_events_created_at ON rate_limit_events(created_at);
CREATE INDEX IF NOT EXISTS idx_rate_limit_status_client_resource ON rate_limit_status(client_id, resource);
CREATE INDEX IF NOT EXISTS idx_rules_type_enabled ON rules(rule_type, enabled);
CREATE INDEX IF NOT EXISTS idx_rules_priority ON rules(priority DESC);
CREATE INDEX IF NOT EXISTS idx_rule_evaluation_history_rule_id ON rule_evaluation_history(rule_id);
CREATE INDEX IF NOT EXISTS idx_rule_evaluation_history_client_id ON rule_evaluation_history(client_id);
CREATE INDEX IF NOT EXISTS idx_rule_evaluation_history_evaluated_at ON rule_evaluation_history(evaluated_at);

-- Insert default rate limiting rules
INSERT INTO rate_limit_rules (id, resource, limit_count, window_duration, algorithm) VALUES
    ('default-api-rule', 'api', 100, INTERVAL '1 minute', 'sliding_window'),
    ('default-login-rule', 'login', 5, INTERVAL '15 minutes', 'fixed_window'),
    ('default-upload-rule', 'upload', 10, INTERVAL '1 hour', 'sliding_window')
ON CONFLICT (id) DO NOTHING;

-- Insert default security rules
INSERT INTO rules (id, name, rule_type, description, priority, enabled, conditions, actions, tags) VALUES
    ('block-suspicious-agents', 'Block Suspicious User Agents', 'blacklist', 'Block requests from suspicious user agents', 200, TRUE,
     '[{"field": "user_agent", "operator": "contains", "value": "bot"}]',
     '[{"type": "deny", "parameters": {"reason": "suspicious user agent"}}]',
     ARRAY['security', 'user-agent']),
    
    ('aggressive-login-rate-limit', 'Aggressive Login Rate Limiting', 'rate_limit', 'Apply stricter rate limiting for login endpoints', 150, TRUE,
     '[{"field": "resource", "operator": "equals", "value": "login"}]',
     '[{"type": "rate_limit", "parameters": {"limit": 3, "window": "5m", "algorithm": "fixed_window"}}]',
     ARRAY['security', 'login']),
    
    ('whitelist-internal-ips', 'Whitelist Internal IPs', 'whitelist', 'Allow all requests from internal IP ranges', 300, TRUE,
     '[{"field": "ip_address", "operator": "starts_with", "value": "192.168."}]',
     '[{"type": "allow", "parameters": {"reason": "internal IP"}}]',
     ARRAY['security', 'whitelist'])
ON CONFLICT (id) DO NOTHING;

-- Create a function to update timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for automatic timestamp updates
CREATE TRIGGER update_rate_limit_rules_updated_at BEFORE UPDATE ON rate_limit_rules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_rate_limit_status_updated_at BEFORE UPDATE ON rate_limit_status FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_rules_updated_at BEFORE UPDATE ON rules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ratelimiter;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ratelimiter;
