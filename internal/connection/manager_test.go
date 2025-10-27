package connection

import (
	"net"
	"testing"
	"time"
)

type mockAddr struct{}

func (m *mockAddr) Network() string { return "tcp" }
func (m *mockAddr) String() string  { return "127.0.0.1:0" }

type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return &mockAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &mockAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestManager_Register(t *testing.T) {
	m := NewManager(10)
	conn := &mockConn{}

	err := m.Register("conn1", "90210", "Beverly Hills", conn)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if m.Count() != 1 {
		t.Errorf("Expected 1 connection, got %d", m.Count())
	}

	client, exists := m.Get("conn1")
	if !exists {
		t.Fatal("Client not found")
	}

	if client.Zipcode != "90210" {
		t.Errorf("Expected zipcode 90210, got %s", client.Zipcode)
	}
}

func TestManager_RegisterMaxConnections(t *testing.T) {
	m := NewManager(2)
	conn := &mockConn{}

	m.Register("conn1", "90210", "Beverly Hills", conn)
	m.Register("conn2", "33139", "Miami Beach", conn)

	// Third connection should fail
	err := m.Register("conn3", "10001", "New York", conn)
	if err != ErrMaxConnectionsReached {
		t.Errorf("Expected ErrMaxConnectionsReached, got %v", err)
	}
}

func TestManager_Unregister(t *testing.T) {
	m := NewManager(10)
	conn := &mockConn{}

	m.Register("conn1", "90210", "Beverly Hills", conn)
	m.Register("conn2", "90210", "Beverly Hills", conn)

	err := m.Unregister("conn1")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	if m.Count() != 1 {
		t.Errorf("Expected 1 connection, got %d", m.Count())
	}

	// Zipcode should still have one connection
	connIDs := m.GetByZipcode("90210")
	if len(connIDs) != 1 {
		t.Errorf("Expected 1 connection for zipcode, got %d", len(connIDs))
	}
}

func TestManager_GetByZipcode(t *testing.T) {
	m := NewManager(10)
	conn := &mockConn{}

	m.Register("conn1", "90210", "Beverly Hills", conn)
	m.Register("conn2", "90210", "Beverly Hills", conn)
	m.Register("conn3", "33139", "Miami Beach", conn)

	connIDs := m.GetByZipcode("90210")
	if len(connIDs) != 2 {
		t.Errorf("Expected 2 connections for 90210, got %d", len(connIDs))
	}

	connIDs = m.GetByZipcode("33139")
	if len(connIDs) != 1 {
		t.Errorf("Expected 1 connection for 33139, got %d", len(connIDs))
	}
}

func TestManager_UpdateActivity(t *testing.T) {
	m := NewManager(10)
	conn := &mockConn{}

	m.Register("conn1", "90210", "Beverly Hills", conn)

	client, _ := m.Get("conn1")
	firstHeard := client.GetLastHeardFrom()

	time.Sleep(10 * time.Millisecond)

	err := m.UpdateActivity("conn1")
	if err != nil {
		t.Fatalf("UpdateActivity failed: %v", err)
	}

	client, _ = m.Get("conn1")
	secondHeard := client.GetLastHeardFrom()

	if !secondHeard.After(firstHeard) {
		t.Error("LastHeardFrom was not updated")
	}
}

func TestManager_GetInactiveConnections(t *testing.T) {
	m := NewManager(10)
	conn := &mockConn{}

	m.Register("conn1", "90210", "Beverly Hills", conn)
	m.Register("conn2", "33139", "Miami Beach", conn)

	// Make conn1 inactive by manually setting its timestamp
	client1, _ := m.Get("conn1")
	client1.mu.Lock()
	client1.LastHeardFrom = time.Now().Add(-5 * time.Minute)
	client1.mu.Unlock()

	inactive := m.GetInactiveConnections(2 * time.Minute)
	if len(inactive) != 1 {
		t.Errorf("Expected 1 inactive connection, got %d", len(inactive))
	}

	if inactive[0] != "conn1" {
		t.Errorf("Expected conn1 to be inactive, got %s", inactive[0])
	}
}

func TestManager_Stats(t *testing.T) {
	m := NewManager(100)
	conn := &mockConn{}

	m.Register("conn1", "90210", "Beverly Hills", conn)
	m.Register("conn2", "90210", "Beverly Hills", conn)
	m.Register("conn3", "33139", "Miami Beach", conn)

	stats := m.Stats()
	if stats.TotalConnections != 3 {
		t.Errorf("Expected 3 connections, got %d", stats.TotalConnections)
	}
	if stats.UniqueZipcodes != 2 {
		t.Errorf("Expected 2 unique zipcodes, got %d", stats.UniqueZipcodes)
	}
	if stats.MaxConnections != 100 {
		t.Errorf("Expected max 100, got %d", stats.MaxConnections)
	}
}
