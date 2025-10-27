-- Weather Server Database Schema
-- Migration 001: Initial Schema

-- Locations table stores information about weather monitoring locations
CREATE TABLE IF NOT EXISTS locations (
    zipcode VARCHAR(10) PRIMARY KEY,
    city_name VARCHAR(255) NOT NULL,
    lat DECIMAL(10, 7),
    lon DECIMAL(10, 7),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_locations_city ON locations(city_name);

-- Raw metrics table stores 5-minute weather data
CREATE TABLE IF NOT EXISTS raw_metrics (
    id BIGSERIAL PRIMARY KEY,
    zipcode VARCHAR(10) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    temperature DECIMAL(5, 2),
    humidity DECIMAL(5, 2),
    precipitation DECIMAL(5, 2),
    wind_speed DECIMAL(5, 2),
    wind_direction VARCHAR(3),
    pollution_index DECIMAL(5, 2),
    pollen_index DECIMAL(5, 2),
    received_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (zipcode) REFERENCES locations(zipcode) ON DELETE CASCADE
);

CREATE INDEX idx_raw_metrics_zipcode_timestamp ON raw_metrics(zipcode, timestamp);
CREATE INDEX idx_raw_metrics_timestamp ON raw_metrics(timestamp);

-- Hourly metrics table stores hourly aggregated data
CREATE TABLE IF NOT EXISTS hourly_metrics (
    id BIGSERIAL PRIMARY KEY,
    zipcode VARCHAR(10) NOT NULL,
    hour_timestamp TIMESTAMPTZ NOT NULL,
    avg_temp DECIMAL(5, 2),
    avg_humidity DECIMAL(5, 2),
    avg_precip DECIMAL(5, 2),
    avg_wind DECIMAL(5, 2),
    avg_pollution DECIMAL(5, 2),
    avg_pollen DECIMAL(5, 2),
    sample_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (zipcode) REFERENCES locations(zipcode) ON DELETE CASCADE,
    UNIQUE(zipcode, hour_timestamp)
);

CREATE INDEX idx_hourly_metrics_zipcode_hour ON hourly_metrics(zipcode, hour_timestamp);
CREATE INDEX idx_hourly_metrics_hour ON hourly_metrics(hour_timestamp);

-- Daily summary table stores daily min/max data
CREATE TABLE IF NOT EXISTS daily_summary (
    id BIGSERIAL PRIMARY KEY,
    zipcode VARCHAR(10) NOT NULL,
    date DATE NOT NULL,
    min_temp DECIMAL(5, 2),
    max_temp DECIMAL(5, 2),
    min_humidity DECIMAL(5, 2),
    max_humidity DECIMAL(5, 2),
    min_precip DECIMAL(5, 2),
    max_precip DECIMAL(5, 2),
    min_wind DECIMAL(5, 2),
    max_wind DECIMAL(5, 2),
    min_pollution DECIMAL(5, 2),
    max_pollution DECIMAL(5, 2),
    min_pollen DECIMAL(5, 2),
    max_pollen DECIMAL(5, 2),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (zipcode) REFERENCES locations(zipcode) ON DELETE CASCADE,
    UNIQUE(zipcode, date)
);

CREATE INDEX idx_daily_summary_zipcode_date ON daily_summary(zipcode, date);
CREATE INDEX idx_daily_summary_date ON daily_summary(date);

-- Comments for documentation
COMMENT ON TABLE locations IS 'Weather monitoring locations';
COMMENT ON TABLE raw_metrics IS '5-minute interval weather measurements';
COMMENT ON TABLE hourly_metrics IS 'Hourly aggregated weather data';
COMMENT ON TABLE daily_summary IS 'Daily min/max weather statistics';

