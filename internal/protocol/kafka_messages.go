package protocol

import (
	"encoding/json"
	"time"
)

// MetricMessage is the internal message format for Kafka
type MetricMessage struct {
	ConnectionID string     `json:"connection_id"`
	Zipcode      string     `json:"zipcode"`
	City         string     `json:"city"`
	ReceivedAt   time.Time  `json:"received_at"`
	Data         MetricData `json:"data"`
}

// ParsedMetricData contains the metric data with parsed timestamp
type ParsedMetricData struct {
	Timestamp      time.Time
	Temperature    float64
	Humidity       float64
	Precipitation  float64
	WindSpeed      float64
	WindDirection  string
	PollutionIndex float64
	PollenIndex    float64
}

// ParseMetricData converts MetricData to ParsedMetricData
func (m *MetricData) Parse() (*ParsedMetricData, error) {
	ts, err := time.Parse(time.RFC3339, m.Timestamp)
	if err != nil {
		return nil, err
	}

	return &ParsedMetricData{
		Timestamp:      ts,
		Temperature:    m.Temperature,
		Humidity:       m.Humidity,
		Precipitation:  m.Precipitation,
		WindSpeed:      m.WindSpeed,
		WindDirection:  m.WindDirection,
		PollutionIndex: m.PollutionIndex,
		PollenIndex:    m.PollenIndex,
	}, nil
}

// AlarmNotification is the message format for alarm notifications
type AlarmNotification struct {
	Type      string    `json:"type"` // ALARM_TRIGGERED, ALARM_CLEARED
	Zipcode   string    `json:"zipcode"`
	City      string    `json:"city"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Operator  string    `json:"operator"`
	Duration  int       `json:"duration_minutes"`
	StartTime time.Time `json:"start_time"`
	AlarmID   int64     `json:"alarm_id,omitempty"`
}

const (
	AlarmTypeTriggered = "ALARM_TRIGGERED"
	AlarmTypeCleared   = "ALARM_CLEARED"
)

// EncodeMetricMessage encodes a MetricMessage to JSON
func EncodeMetricMessage(msg *MetricMessage) ([]byte, error) {
	return json.Marshal(msg)
}

// DecodeMetricMessage decodes JSON to MetricMessage
func DecodeMetricMessage(data []byte) (*MetricMessage, error) {
	var msg MetricMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// EncodeAlarmNotification encodes an AlarmNotification to JSON
func EncodeAlarmNotification(alarm *AlarmNotification) ([]byte, error) {
	return json.Marshal(alarm)
}

// DecodeAlarmNotification decodes JSON to AlarmNotification
func DecodeAlarmNotification(data []byte) (*AlarmNotification, error) {
	var alarm AlarmNotification
	if err := json.Unmarshal(data, &alarm); err != nil {
		return nil, err
	}
	return &alarm, nil
}
