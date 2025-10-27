package alarming

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/smukkama/weather-server/internal/database"
	"github.com/smukkama/weather-server/internal/protocol"
	"github.com/smukkama/weather-server/internal/queue"
)

// Evaluator evaluates metrics against thresholds and manages alarm state
type Evaluator struct {
	db             *database.DB
	stateManager   *StateManager
	alarmProducer  *queue.Producer
	thresholdCache map[string][]*database.AlarmThreshold
	lastCacheLoad  time.Time
	cacheValidity  time.Duration
}

// NewEvaluator creates a new alarm evaluator
func NewEvaluator(db *database.DB, stateManager *StateManager, alarmProducer *queue.Producer) *Evaluator {
	return &Evaluator{
		db:             db,
		stateManager:   stateManager,
		alarmProducer:  alarmProducer,
		thresholdCache: make(map[string][]*database.AlarmThreshold),
		cacheValidity:  5 * time.Minute,
	}
}

// EvaluateMetric evaluates a metric message against all thresholds
func (e *Evaluator) EvaluateMetric(ctx context.Context, msg *protocol.MetricMessage) error {
	// Parse metric data
	parsedData, err := msg.Data.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse metric data: %w", err)
	}

	// Get thresholds for this zipcode
	thresholds, err := e.getThresholds(msg.Zipcode)
	if err != nil {
		return fmt.Errorf("failed to get thresholds: %w", err)
	}

	// Evaluate each threshold
	for _, threshold := range thresholds {
		value := e.extractMetricValue(parsedData, threshold.MetricName)
		if value == nil {
			continue
		}

		if err := e.evaluateThreshold(ctx, msg, threshold, *value); err != nil {
			fmt.Printf("Failed to evaluate threshold: %v\n", err)
		}
	}

	return nil
}

func (e *Evaluator) evaluateThreshold(ctx context.Context, msg *protocol.MetricMessage, threshold *database.AlarmThreshold, value float64) error {
	// Check if threshold is breached
	breached := evaluateCondition(value, threshold.Operator, threshold.ThresholdValue)

	// Get current state
	state, err := e.stateManager.GetState(ctx, msg.Zipcode, threshold.MetricName)
	if err != nil {
		return err
	}

	now := time.Now()

	if breached {
		return e.handleBreach(ctx, msg, threshold, value, state, now)
	} else {
		return e.handleNoBreach(ctx, msg, threshold, state, now)
	}
}

func (e *Evaluator) handleBreach(ctx context.Context, msg *protocol.MetricMessage, threshold *database.AlarmThreshold, value float64, state *AlarmState, now time.Time) error {
	switch state.Status {
	case AlarmStateClear:
		// New breach detected
		newState := &AlarmState{
			Status:          AlarmStatePending,
			BreachStartTime: now,
			LastChecked:     now,
			BreachValue:     value,
		}
		return e.stateManager.SetState(ctx, msg.Zipcode, threshold.MetricName, newState)

	case AlarmStatePending:
		// Check if duration met
		durationMet := now.Sub(state.BreachStartTime) >= time.Duration(threshold.DurationMinutes)*time.Minute

		if durationMet {
			// TRIGGER ALARM
			return e.triggerAlarm(ctx, msg, threshold, value, state, now)
		}

		// Update last checked
		state.LastChecked = now
		state.BreachValue = value
		return e.stateManager.SetState(ctx, msg.Zipcode, threshold.MetricName, state)

	case AlarmStateActive:
		// Alarm already active, update last checked
		state.LastChecked = now
		return e.stateManager.SetState(ctx, msg.Zipcode, threshold.MetricName, state)
	}

	return nil
}

func (e *Evaluator) handleNoBreach(ctx context.Context, msg *protocol.MetricMessage, threshold *database.AlarmThreshold, state *AlarmState, now time.Time) error {
	switch state.Status {
	case AlarmStateClear:
		// Nothing to do
		return nil

	case AlarmStatePending:
		// Breach ended before alarm triggered
		return e.stateManager.DeleteState(ctx, msg.Zipcode, threshold.MetricName)

	case AlarmStateActive:
		// CLEAR ALARM
		return e.clearAlarm(ctx, msg, threshold, state, now)
	}

	return nil
}

func (e *Evaluator) triggerAlarm(ctx context.Context, msg *protocol.MetricMessage, threshold *database.AlarmThreshold, value float64, state *AlarmState, now time.Time) error {
	fmt.Printf("🚨 ALARM TRIGGERED: %s (zipcode=%s, metric=%s, value=%.2f, threshold=%.2f)\n",
		msg.City, msg.Zipcode, threshold.MetricName, value, threshold.ThresholdValue)

	// Create alarm log entry
	thresholdConfig, _ := json.Marshal(threshold)
	alarmLog := &database.AlarmLog{
		Zipcode:         msg.Zipcode,
		MetricName:      threshold.MetricName,
		BreachValue:     value,
		ThresholdConfig: string(thresholdConfig),
		StartTime:       state.BreachStartTime,
		Status:          database.AlarmStatusActive,
	}

	if err := e.db.InsertAlarmLog(alarmLog); err != nil {
		return fmt.Errorf("failed to insert alarm log: %w", err)
	}

	// Update state to ALARMING
	state.Status = AlarmStateActive
	state.AlarmID = alarmLog.AlarmID
	state.LastChecked = now
	if err := e.stateManager.SetState(ctx, msg.Zipcode, threshold.MetricName, state); err != nil {
		return err
	}

	// Send notification
	notification := &protocol.AlarmNotification{
		Type:      protocol.AlarmTypeTriggered,
		Zipcode:   msg.Zipcode,
		City:      msg.City,
		Metric:    threshold.MetricName,
		Value:     value,
		Threshold: threshold.ThresholdValue,
		Operator:  threshold.Operator,
		Duration:  threshold.DurationMinutes,
		StartTime: state.BreachStartTime,
		AlarmID:   alarmLog.AlarmID,
	}

	return e.sendNotification(ctx, notification)
}

func (e *Evaluator) clearAlarm(ctx context.Context, msg *protocol.MetricMessage, threshold *database.AlarmThreshold, state *AlarmState, now time.Time) error {
	fmt.Printf("✅ ALARM CLEARED: %s (zipcode=%s, metric=%s)\n",
		msg.City, msg.Zipcode, threshold.MetricName)

	// Update alarm log
	if state.AlarmID > 0 {
		if err := e.db.UpdateAlarmLogCleared(state.AlarmID, now); err != nil {
			return fmt.Errorf("failed to update alarm log: %w", err)
		}
	}

	// Delete state
	if err := e.stateManager.DeleteState(ctx, msg.Zipcode, threshold.MetricName); err != nil {
		return err
	}

	// Send clear notification
	notification := &protocol.AlarmNotification{
		Type:      protocol.AlarmTypeCleared,
		Zipcode:   msg.Zipcode,
		City:      msg.City,
		Metric:    threshold.MetricName,
		Threshold: threshold.ThresholdValue,
		AlarmID:   state.AlarmID,
	}

	return e.sendNotification(ctx, notification)
}

func (e *Evaluator) sendNotification(ctx context.Context, notification *protocol.AlarmNotification) error {
	data, err := protocol.EncodeAlarmNotification(notification)
	if err != nil {
		return fmt.Errorf("failed to encode notification: %w", err)
	}

	key := fmt.Sprintf("%s-%s", notification.Zipcode, notification.Metric)
	return e.alarmProducer.Publish(ctx, key, data)
}

func (e *Evaluator) getThresholds(zipcode string) ([]*database.AlarmThreshold, error) {
	// Check cache
	if time.Since(e.lastCacheLoad) < e.cacheValidity {
		if thresholds, ok := e.thresholdCache[zipcode]; ok {
			return thresholds, nil
		}
	}

	// Load from database
	thresholds, err := e.db.GetActiveAlarmThresholds(zipcode)
	if err != nil {
		return nil, err
	}

	e.thresholdCache[zipcode] = thresholds
	e.lastCacheLoad = time.Now()

	return thresholds, nil
}

func (e *Evaluator) extractMetricValue(data *protocol.ParsedMetricData, metricName string) *float64 {
	switch metricName {
	case "temperature":
		return &data.Temperature
	case "humidity":
		return &data.Humidity
	case "precipitation":
		return &data.Precipitation
	case "wind_speed":
		return &data.WindSpeed
	case "pollution_index":
		return &data.PollutionIndex
	case "pollen_index":
		return &data.PollenIndex
	default:
		return nil
	}
}

func evaluateCondition(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	default:
		return false
	}
}
