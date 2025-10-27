-- Weather Server Database Schema
-- Migration 002: Alarm Tables

-- Alarm thresholds table stores alarm configuration
CREATE TABLE IF NOT EXISTS alarm_thresholds (
    id SERIAL PRIMARY KEY,
    zipcode VARCHAR(10) NOT NULL,
    metric_name VARCHAR(50) NOT NULL,
    operator VARCHAR(2) NOT NULL CHECK (operator IN ('>', '<', '>=', '<=')),
    threshold_value DECIMAL(10, 2) NOT NULL,
    duration_minutes INTEGER NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (zipcode) REFERENCES locations(zipcode) ON DELETE CASCADE,
    UNIQUE(zipcode, metric_name)
);

CREATE INDEX idx_alarm_thresholds_zipcode ON alarm_thresholds(zipcode);
CREATE INDEX idx_alarm_thresholds_active ON alarm_thresholds(is_active) WHERE is_active = true;

-- Alarms log table stores alarm history
CREATE TABLE IF NOT EXISTS alarms_log (
    alarm_id BIGSERIAL PRIMARY KEY,
    zipcode VARCHAR(10) NOT NULL,
    metric_name VARCHAR(50) NOT NULL,
    breach_value DECIMAL(10, 2) NOT NULL,
    threshold_config JSONB NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL CHECK (status IN ('ACTIVE', 'CLEARED')),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (zipcode) REFERENCES locations(zipcode) ON DELETE CASCADE
);

CREATE INDEX idx_alarms_log_zipcode ON alarms_log(zipcode);
CREATE INDEX idx_alarms_log_status ON alarms_log(status);
CREATE INDEX idx_alarms_log_start_time ON alarms_log(start_time);
CREATE INDEX idx_alarms_log_zipcode_status ON alarms_log(zipcode, status);

-- Comments for documentation
COMMENT ON TABLE alarm_thresholds IS 'Configurable alarm thresholds per location and metric';
COMMENT ON TABLE alarms_log IS 'Historical log of all triggered alarms';
COMMENT ON COLUMN alarm_thresholds.metric_name IS 'Metric name: temperature, humidity, precipitation, wind_speed, pollution_index, pollen_index';
COMMENT ON COLUMN alarm_thresholds.duration_minutes IS 'Duration the threshold must be breached before triggering alarm';
COMMENT ON COLUMN alarms_log.threshold_config IS 'JSON snapshot of the threshold configuration at time of alarm';

