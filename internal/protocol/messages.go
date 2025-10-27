package protocol

import (
	"encoding/json"
	"fmt"
	"time"
)

// MessageType represents the type of message
type MessageType string

const (
	// Client to Server
	MsgTypeIdentify  MessageType = "identify"
	MsgTypeMetrics   MessageType = "metrics"
	MsgTypeKeepalive MessageType = "keepalive"

	// Server to Client
	MsgTypeAck MessageType = "ack"
)

// BaseMessage is the common structure for all messages
type BaseMessage struct {
	Type MessageType `json:"type"`
}

// IdentifyMessage is sent by the client on connection
type IdentifyMessage struct {
	Type    MessageType `json:"type"`
	Zipcode string      `json:"zipcode"`
	City    string      `json:"city"`
}

// MetricData contains the actual weather measurements
type MetricData struct {
	Timestamp      string  `json:"timestamp"`
	Temperature    float64 `json:"temperature"`
	Humidity       float64 `json:"humidity"`
	Precipitation  float64 `json:"precipitation"`
	WindSpeed      float64 `json:"wind_speed"`
	WindDirection  string  `json:"wind_direction"`
	PollutionIndex float64 `json:"pollution_index"`
	PollenIndex    float64 `json:"pollen_index"`
}

// MetricsMessage is sent by the client every 5 minutes
type MetricsMessage struct {
	Type MessageType `json:"type"`
	Data MetricData  `json:"data"`
}

// KeepaliveMessage is sent by the client every 30-60 seconds
type KeepaliveMessage struct {
	Type MessageType `json:"type"`
}

// AckMessage is sent by the server in response to messages
type AckMessage struct {
	Type   MessageType `json:"type"`
	Status string      `json:"status"`
}

// AckStatus constants
const (
	AckStatusIdentified = "identified"
	AckStatusAlive      = "alive"
	AckStatusError      = "error"
)

// ParseMessage parses a JSON line into the appropriate message type
func ParseMessage(data []byte) (interface{}, error) {
	var base BaseMessage
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	switch base.Type {
	case MsgTypeIdentify:
		var msg IdentifyMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("invalid identify message: %w", err)
		}
		if err := validateIdentify(&msg); err != nil {
			return nil, err
		}
		return &msg, nil

	case MsgTypeMetrics:
		var msg MetricsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("invalid metrics message: %w", err)
		}
		if err := validateMetrics(&msg); err != nil {
			return nil, err
		}
		return &msg, nil

	case MsgTypeKeepalive:
		var msg KeepaliveMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return nil, fmt.Errorf("invalid keepalive message: %w", err)
		}
		return &msg, nil

	default:
		return nil, fmt.Errorf("unknown message type: %s", base.Type)
	}
}

// validateIdentify validates an identify message
func validateIdentify(msg *IdentifyMessage) error {
	if msg.Zipcode == "" {
		return fmt.Errorf("zipcode is required")
	}
	if msg.City == "" {
		return fmt.Errorf("city is required")
	}
	return nil
}

// validateMetrics validates a metrics message
func validateMetrics(msg *MetricsMessage) error {
	if msg.Data.Timestamp == "" {
		return fmt.Errorf("timestamp is required")
	}
	// Validate timestamp format
	if _, err := time.Parse(time.RFC3339, msg.Data.Timestamp); err != nil {
		return fmt.Errorf("invalid timestamp format (must be RFC3339): %w", err)
	}
	return nil
}

// EncodeMessage encodes a message to JSON
func EncodeMessage(msg interface{}) ([]byte, error) {
	return json.Marshal(msg)
}

// NewAckMessage creates a new acknowledgment message
func NewAckMessage(status string) *AckMessage {
	return &AckMessage{
		Type:   MsgTypeAck,
		Status: status,
	}
}
