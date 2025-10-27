package aggregation

import (
	"fmt"
	"time"

	"github.com/smukkama/weather-server/internal/database"
)

// HourlyAggregator performs hourly aggregation
type HourlyAggregator struct {
	db *database.DB
}

// NewHourlyAggregator creates a new hourly aggregator
func NewHourlyAggregator(db *database.DB) *HourlyAggregator {
	return &HourlyAggregator{db: db}
}

// Aggregate performs hourly aggregation for the specified hour
func (h *HourlyAggregator) Aggregate(targetHour time.Time) error {
	// Truncate to the beginning of the hour
	startTime := targetHour.Truncate(time.Hour)
	endTime := startTime.Add(time.Hour)

	fmt.Printf("Running hourly aggregation for %s\n", startTime.Format("2006-01-02 15:04:05"))

	query := `
		INSERT INTO hourly_metrics (
			zipcode, hour_timestamp, avg_temp, avg_humidity, avg_precip,
			avg_wind, avg_pollution, avg_pollen, sample_count
		)
		SELECT
			zipcode,
			$1 AS hour_timestamp,
			AVG(temperature) AS avg_temp,
			AVG(humidity) AS avg_humidity,
			AVG(precipitation) AS avg_precip,
			AVG(wind_speed) AS avg_wind,
			AVG(pollution_index) AS avg_pollution,
			AVG(pollen_index) AS avg_pollen,
			COUNT(*) AS sample_count
		FROM
			raw_metrics
		WHERE
			timestamp >= $1 AND timestamp < $2
		GROUP BY
			zipcode
		ON CONFLICT (zipcode, hour_timestamp) DO UPDATE
		SET
			avg_temp = EXCLUDED.avg_temp,
			avg_humidity = EXCLUDED.avg_humidity,
			avg_precip = EXCLUDED.avg_precip,
			avg_wind = EXCLUDED.avg_wind,
			avg_pollution = EXCLUDED.avg_pollution,
			avg_pollen = EXCLUDED.avg_pollen,
			sample_count = EXCLUDED.sample_count
	`

	result, err := h.db.Exec(query, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to aggregate hourly data: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Hourly aggregation completed: %d zipcodes processed\n", rowsAffected)

	return nil
}

// AggregatePreviousHour aggregates the previous full hour
func (h *HourlyAggregator) AggregatePreviousHour() error {
	now := time.Now()
	previousHour := now.Add(-1 * time.Hour).Truncate(time.Hour)
	return h.Aggregate(previousHour)
}

// CalculateNextRunTime calculates when the hourly aggregation should next run
// It runs at HH:05:00 (5 minutes past each hour)
func (h *HourlyAggregator) CalculateNextRunTime(delay time.Duration) time.Time {
	now := time.Now()

	// Next hour
	nextHour := now.Truncate(time.Hour).Add(time.Hour)

	// Add delay (e.g., 5 minutes past the hour)
	nextRun := nextHour.Add(delay)

	// If we're past the next run time, add another hour
	if now.After(nextRun) {
		nextRun = nextRun.Add(time.Hour)
	}

	return nextRun
}
