package database

import (
	"time"
)

// Location represents a weather monitoring location
type Location struct {
	Zipcode   string
	CityName  string
	Lat       *float64
	Lon       *float64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// RawMetric represents a 5-minute weather measurement
type RawMetric struct {
	ID             int64
	Zipcode        string
	Timestamp      time.Time
	Temperature    *float64
	Humidity       *float64
	Precipitation  *float64
	WindSpeed      *float64
	WindDirection  *string
	PollutionIndex *float64
	PollenIndex    *float64
	ReceivedAt     time.Time
}

// HourlyMetric represents hourly aggregated data
type HourlyMetric struct {
	ID            int64
	Zipcode       string
	HourTimestamp time.Time
	AvgTemp       *float64
	AvgHumidity   *float64
	AvgPrecip     *float64
	AvgWind       *float64
	AvgPollution  *float64
	AvgPollen     *float64
	SampleCount   int
	CreatedAt     time.Time
}

// DailySummary represents daily min/max data
type DailySummary struct {
	ID           int64
	Zipcode      string
	Date         time.Time
	MinTemp      *float64
	MaxTemp      *float64
	MinHumidity  *float64
	MaxHumidity  *float64
	MinPrecip    *float64
	MaxPrecip    *float64
	MinWind      *float64
	MaxWind      *float64
	MinPollution *float64
	MaxPollution *float64
	MinPollen    *float64
	MaxPollen    *float64
	CreatedAt    time.Time
}

// AlarmThreshold represents an alarm configuration
type AlarmThreshold struct {
	ID              int
	Zipcode         string
	MetricName      string
	Operator        string
	ThresholdValue  float64
	DurationMinutes int
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// AlarmLog represents a logged alarm event
type AlarmLog struct {
	AlarmID         int64
	Zipcode         string
	MetricName      string
	BreachValue     float64
	ThresholdConfig string // JSON
	StartTime       time.Time
	EndTime         *time.Time
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

const (
	AlarmStatusActive  = "ACTIVE"
	AlarmStatusCleared = "CLEARED"
)
