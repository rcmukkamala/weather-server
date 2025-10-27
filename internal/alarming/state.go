package alarming

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// AlarmState represents the current state of an alarm
type AlarmState struct {
	Status          string    `json:"status"` // CLEAR, PENDING_ALARM, ALARMING
	BreachStartTime time.Time `json:"breach_start_time"`
	LastChecked     time.Time `json:"last_checked"`
	BreachValue     float64   `json:"breach_value"`
	AlarmID         int64     `json:"alarm_id,omitempty"`
}

const (
	AlarmStateClear   = "CLEAR"
	AlarmStatePending = "PENDING_ALARM"
	AlarmStateActive  = "ALARMING"
)

// StateManager manages alarm states in Redis
type StateManager struct {
	redis *redis.Client
}

// NewStateManager creates a new state manager
func NewStateManager(redisClient *redis.Client) *StateManager {
	return &StateManager{redis: redisClient}
}

// GetState retrieves the alarm state for a location and metric
func (sm *StateManager) GetState(ctx context.Context, zipcode, metric string) (*AlarmState, error) {
	key := fmt.Sprintf("alarm_state:%s:%s", zipcode, metric)

	data, err := sm.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		// No state exists, return CLEAR state
		return &AlarmState{
			Status: AlarmStateClear,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get state from Redis: %w", err)
	}

	var state AlarmState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// SetState saves the alarm state for a location and metric
func (sm *StateManager) SetState(ctx context.Context, zipcode, metric string, state *AlarmState) error {
	key := fmt.Sprintf("alarm_state:%s:%s", zipcode, metric)

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Set with expiration (e.g., 7 days) to auto-cleanup stale states
	if err := sm.redis.Set(ctx, key, data, 7*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to set state in Redis: %w", err)
	}

	return nil
}

// DeleteState removes the alarm state (returns to CLEAR)
func (sm *StateManager) DeleteState(ctx context.Context, zipcode, metric string) error {
	key := fmt.Sprintf("alarm_state:%s:%s", zipcode, metric)
	return sm.redis.Del(ctx, key).Err()
}

// GetAllStates returns all active alarm states (for monitoring)
func (sm *StateManager) GetAllStates(ctx context.Context) (map[string]*AlarmState, error) {
	pattern := "alarm_state:*"

	keys, err := sm.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	states := make(map[string]*AlarmState)
	for _, key := range keys {
		data, err := sm.redis.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var state AlarmState
		if err := json.Unmarshal([]byte(data), &state); err != nil {
			continue
		}

		states[key] = &state
	}

	return states, nil
}
