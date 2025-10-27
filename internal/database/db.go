package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps the database connection
type DB struct {
	*sql.DB
}

// Connect establishes a connection to the database
func Connect(connectionString string) (*DB, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return &DB{db}, nil
}

// RunMigrations executes all SQL migration files in order
func (db *DB) RunMigrations(migrationsDir string) error {
	// Read all migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort SQL files
	var sqlFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file.Name())
		}
	}
	sort.Strings(sqlFiles)

	// Execute each migration
	for _, filename := range sqlFiles {
		fmt.Printf("Running migration: %s\n", filename)

		filePath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}
	}

	fmt.Println("All migrations completed successfully")
	return nil
}

// UpsertLocation inserts or updates a location
func (db *DB) UpsertLocation(loc *Location) error {
	query := `
		INSERT INTO locations (zipcode, city_name, lat, lon)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (zipcode) DO UPDATE
		SET city_name = EXCLUDED.city_name,
		    lat = EXCLUDED.lat,
		    lon = EXCLUDED.lon,
		    updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.Exec(query, loc.Zipcode, loc.CityName, loc.Lat, loc.Lon)
	return err
}

// GetLocation retrieves a location by zipcode
func (db *DB) GetLocation(zipcode string) (*Location, error) {
	query := `
		SELECT zipcode, city_name, lat, lon, created_at, updated_at
		FROM locations
		WHERE zipcode = $1
	`

	var loc Location
	err := db.QueryRow(query, zipcode).Scan(
		&loc.Zipcode,
		&loc.CityName,
		&loc.Lat,
		&loc.Lon,
		&loc.CreatedAt,
		&loc.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &loc, nil
}

// InsertRawMetric inserts a raw weather metric
func (db *DB) InsertRawMetric(metric *RawMetric) error {
	query := `
		INSERT INTO raw_metrics (
			zipcode, timestamp, temperature, humidity, precipitation,
			wind_speed, wind_direction, pollution_index, pollen_index, received_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	return db.QueryRow(
		query,
		metric.Zipcode,
		metric.Timestamp,
		metric.Temperature,
		metric.Humidity,
		metric.Precipitation,
		metric.WindSpeed,
		metric.WindDirection,
		metric.PollutionIndex,
		metric.PollenIndex,
		metric.ReceivedAt,
	).Scan(&metric.ID)
}

// GetActiveAlarmThresholds retrieves all active alarm thresholds for a zipcode
func (db *DB) GetActiveAlarmThresholds(zipcode string) ([]*AlarmThreshold, error) {
	query := `
		SELECT id, zipcode, metric_name, operator, threshold_value,
		       duration_minutes, is_active, created_at, updated_at
		FROM alarm_thresholds
		WHERE zipcode = $1 AND is_active = true
		ORDER BY metric_name
	`

	rows, err := db.Query(query, zipcode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var thresholds []*AlarmThreshold
	for rows.Next() {
		var t AlarmThreshold
		if err := rows.Scan(
			&t.ID,
			&t.Zipcode,
			&t.MetricName,
			&t.Operator,
			&t.ThresholdValue,
			&t.DurationMinutes,
			&t.IsActive,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		thresholds = append(thresholds, &t)
	}

	return thresholds, rows.Err()
}

// InsertAlarmLog inserts a new alarm log entry
func (db *DB) InsertAlarmLog(alarm *AlarmLog) error {
	query := `
		INSERT INTO alarms_log (
			zipcode, metric_name, breach_value, threshold_config,
			start_time, status
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING alarm_id
	`

	return db.QueryRow(
		query,
		alarm.Zipcode,
		alarm.MetricName,
		alarm.BreachValue,
		alarm.ThresholdConfig,
		alarm.StartTime,
		alarm.Status,
	).Scan(&alarm.AlarmID)
}

// UpdateAlarmLogCleared updates an alarm log to cleared status
func (db *DB) UpdateAlarmLogCleared(alarmID int64, endTime time.Time) error {
	query := `
		UPDATE alarms_log
		SET status = $1, end_time = $2, updated_at = CURRENT_TIMESTAMP
		WHERE alarm_id = $3
	`

	_, err := db.Exec(query, AlarmStatusCleared, endTime, alarmID)
	return err
}
