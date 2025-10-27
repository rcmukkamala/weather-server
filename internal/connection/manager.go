package connection

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// ClientInfo holds information about a connected client
type ClientInfo struct {
	ConnectionID  string
	Zipcode       string
	City          string
	ConnectedAt   time.Time
	LastHeardFrom time.Time
	Conn          net.Conn
	mu            sync.RWMutex
}

// UpdateLastHeardFrom updates the last activity timestamp
func (c *ClientInfo) UpdateLastHeardFrom() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastHeardFrom = time.Now()
}

// GetLastHeardFrom returns the last activity timestamp
func (c *ClientInfo) GetLastHeardFrom() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastHeardFrom
}

// Manager manages all active client connections
type Manager struct {
	clients   map[string]*ClientInfo // key: connection_id
	byZipcode map[string][]string    // key: zipcode, value: []connection_id
	mu        sync.RWMutex
	maxConns  int
}

// NewManager creates a new connection manager
func NewManager(maxConnections int) *Manager {
	return &Manager{
		clients:   make(map[string]*ClientInfo),
		byZipcode: make(map[string][]string),
		maxConns:  maxConnections,
	}
}

// Register adds a new client connection
func (m *Manager) Register(connectionID, zipcode, city string, conn net.Conn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check max connections
	if len(m.clients) >= m.maxConns {
		return ErrMaxConnectionsReached
	}

	// Check if connection ID already exists
	if _, exists := m.clients[connectionID]; exists {
		return fmt.Errorf("connection ID %s already registered", connectionID)
	}

	now := time.Now()
	clientInfo := &ClientInfo{
		ConnectionID:  connectionID,
		Zipcode:       zipcode,
		City:          city,
		ConnectedAt:   now,
		LastHeardFrom: now,
		Conn:          conn,
	}

	m.clients[connectionID] = clientInfo
	m.byZipcode[zipcode] = append(m.byZipcode[zipcode], connectionID)

	return nil
}

// Unregister removes a client connection
func (m *Manager) Unregister(connectionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[connectionID]
	if !exists {
		return fmt.Errorf("connection ID %s not found", connectionID)
	}

	// Remove from zipcode map
	zipcode := client.Zipcode
	if connIDs, ok := m.byZipcode[zipcode]; ok {
		// Remove this connection ID from the slice
		for i, id := range connIDs {
			if id == connectionID {
				m.byZipcode[zipcode] = append(connIDs[:i], connIDs[i+1:]...)
				break
			}
		}
		// Clean up empty zipcode entries
		if len(m.byZipcode[zipcode]) == 0 {
			delete(m.byZipcode, zipcode)
		}
	}

	// Remove from clients map
	delete(m.clients, connectionID)

	return nil
}

// Get retrieves client information by connection ID
func (m *Manager) Get(connectionID string) (*ClientInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[connectionID]
	return client, exists
}

// GetByZipcode retrieves all connection IDs for a zipcode
func (m *Manager) GetByZipcode(zipcode string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connIDs := m.byZipcode[zipcode]
	// Return a copy to avoid race conditions
	result := make([]string, len(connIDs))
	copy(result, connIDs)
	return result
}

// UpdateActivity updates the last heard from timestamp for a connection
func (m *Manager) UpdateActivity(connectionID string) error {
	m.mu.RLock()
	client, exists := m.clients[connectionID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection ID %s not found", connectionID)
	}

	client.UpdateLastHeardFrom()
	return nil
}

// GetInactiveConnections returns connection IDs that haven't been heard from in the given duration
func (m *Manager) GetInactiveConnections(timeout time.Duration) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	var inactive []string

	for connID, client := range m.clients {
		lastHeard := client.GetLastHeardFrom()
		if now.Sub(lastHeard) > timeout {
			inactive = append(inactive, connID)
		}
	}

	return inactive
}

// Count returns the total number of active connections
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// CountByZipcode returns the number of active connections per zipcode
func (m *Manager) CountByZipcode() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]int)
	for zipcode, connIDs := range m.byZipcode {
		result[zipcode] = len(connIDs)
	}
	return result
}

// GetAllConnections returns all connection IDs
func (m *Manager) GetAllConnections() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connIDs := make([]string, 0, len(m.clients))
	for connID := range m.clients {
		connIDs = append(connIDs, connID)
	}
	return connIDs
}

// Stats returns statistics about the connection manager
func (m *Manager) Stats() ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return ManagerStats{
		TotalConnections: len(m.clients),
		UniqueZipcodes:   len(m.byZipcode),
		MaxConnections:   m.maxConns,
	}
}

// ManagerStats contains statistics about the connection manager
type ManagerStats struct {
	TotalConnections int
	UniqueZipcodes   int
	MaxConnections   int
}

var (
	ErrMaxConnectionsReached = &ConnectionError{"maximum connections reached"}
)

// ConnectionError represents a connection error
type ConnectionError struct {
	msg string
}

func (e *ConnectionError) Error() string {
	return e.msg
}
