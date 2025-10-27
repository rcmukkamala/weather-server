-- Sample alarm thresholds for testing
-- Run this after the server has created some locations

-- Miami Beach: Hurricane warning (wind > 74 mph for 15 minutes)
INSERT INTO alarm_thresholds (zipcode, metric_name, operator, threshold_value, duration_minutes, is_active)
VALUES ('33139', 'wind_speed', '>', 74.0, 15, true)
ON CONFLICT (zipcode, metric_name) DO NOTHING;

-- Minneapolis: Extreme cold warning (temp < -20°C for 60 minutes)
INSERT INTO alarm_thresholds (zipcode, metric_name, operator, threshold_value, duration_minutes, is_active)
VALUES ('55401', 'temperature', '<', -20.0, 60, true)
ON CONFLICT (zipcode, metric_name) DO NOTHING;

-- Beverly Hills: High pollution alert (pollution > 150 for 30 minutes)
INSERT INTO alarm_thresholds (zipcode, metric_name, operator, threshold_value, duration_minutes, is_active)
VALUES ('90210', 'pollution_index', '>', 150.0, 30, true)
ON CONFLICT (zipcode, metric_name) DO NOTHING;

-- San Francisco: Heavy rain warning (precipitation > 50mm for 60 minutes)
INSERT INTO alarm_thresholds (zipcode, metric_name, operator, threshold_value, duration_minutes, is_active)
VALUES ('94102', 'precipitation', '>', 50.0, 60, true)
ON CONFLICT (zipcode, metric_name) DO NOTHING;

-- Phoenix: Extreme heat warning (temp > 45°C for 120 minutes)
INSERT INTO alarm_thresholds (zipcode, metric_name, operator, threshold_value, duration_minutes, is_active)
VALUES ('85001', 'temperature', '>', 45.0, 120, true)
ON CONFLICT (zipcode, metric_name) DO NOTHING;

-- Seattle: High pollen alert (pollen > 200 for 180 minutes)
INSERT INTO alarm_thresholds (zipcode, metric_name, operator, threshold_value, duration_minutes, is_active)
VALUES ('98101', 'pollen_index', '>', 200.0, 180, true)
ON CONFLICT (zipcode, metric_name) DO NOTHING;

-- Verify insertion
SELECT 
    zipcode, 
    metric_name, 
    operator || ' ' || threshold_value as threshold,
    duration_minutes,
    is_active
FROM alarm_thresholds
ORDER BY zipcode, metric_name;

