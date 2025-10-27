package aggregation

import (
	"fmt"
	"time"

	"github.com/smukkama/weather-server/internal/database"
)

// DailyAggregator performs daily aggregation
type DailyAggregator struct {
	db *database.DB
}

// NewDailyAggregator creates a new daily aggregator
func NewDailyAggregator(db *database.DB) *DailyAggregator {
	return &DailyAggregator{db: db}
}

// Aggregate performs daily aggregation for the specified date
func (d *DailyAggregator) Aggregate(targetDate time.Time) error {
	// Truncate to beginning of day
	date := targetDate.Truncate(24 * time.Hour)

	fmt.Printf("Running daily aggregation for %s\n", date.Format("2006-01-02"))

	query := `
		INSERT INTO daily_summary (
			zipcode, date,
			min_temp, max_temp,
			min_humidity, max_humidity,
			min_precip, max_precip,
			min_wind, max_wind,
			min_pollution, max_pollution,
			min_pollen, max_pollen
		)
		SELECT
			zipcode,
			$1::date AS date,
			MIN(avg_temp) AS min_temp,
			MAX(avg_temp) AS max_temp,
			MIN(avg_humidity) AS min_humidity,
			MAX(avg_humidity) AS max_humidity,
			MIN(avg_precip) AS min_precip,
			MAX(avg_precip) AS max_precip,
			MIN(avg_wind) AS min_wind,
			MAX(avg_wind) AS max_wind,
			MIN(avg_pollution) AS min_pollution,
			MAX(avg_pollution) AS max_pollution,
			MIN(avg_pollen) AS min_pollen,
			MAX(avg_pollen) AS max_pollen
		FROM
			hourly_metrics
		WHERE
			DATE(hour_timestamp) = $1::date
		GROUP BY
			zipcode
		ON CONFLICT (zipcode, date) DO UPDATE
		SET
			min_temp = EXCLUDED.min_temp,
			max_temp = EXCLUDED.max_temp,
			min_humidity = EXCLUDED.min_humidity,
			max_humidity = EXCLUDED.max_humidity,
			min_precip = EXCLUDED.min_precip,
			max_precip = EXCLUDED.max_precip,
			min_wind = EXCLUDED.min_wind,
			max_wind = EXCLUDED.max_wind,
			min_pollution = EXCLUDED.min_pollution,
			max_pollution = EXCLUDED.max_pollution,
			min_pollen = EXCLUDED.min_pollen,
			max_pollen = EXCLUDED.max_pollen
	`

	result, err := d.db.Exec(query, date)
	if err != nil {
		return fmt.Errorf("failed to aggregate daily data: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Daily aggregation completed: %d zipcodes processed\n", rowsAffected)

	return nil
}

// AggregatePreviousDay aggregates the previous full day
func (d *DailyAggregator) AggregatePreviousDay() error {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
	return d.Aggregate(yesterday)
}

// CalculateNextRunTime calculates when the daily aggregation should next run
// It runs at a specific time each day (e.g., 00:05:00)
func (d *DailyAggregator) CalculateNextRunTime(timeOfDay string) (time.Time, error) {
	now := time.Now()

	// Parse time of day (format: "HH:MM")
	var hour, minute int
	if _, err := fmt.Sscanf(timeOfDay, "%d:%d", &hour, &minute); err != nil {
		return time.Time{}, fmt.Errorf("invalid time format: %s (expected HH:MM)", timeOfDay)
	}

	// Today's run time
	todayRun := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	// If we're past today's run time, schedule for tomorrow
	if now.After(todayRun) {
		return todayRun.AddDate(0, 0, 1), nil
	}

	return todayRun, nil
}
