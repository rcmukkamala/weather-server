package server

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smukkama/weather-server/internal/connection"
	"github.com/smukkama/weather-server/internal/protocol"
	"github.com/smukkama/weather-server/internal/queue"
	"github.com/smukkama/weather-server/internal/timer"
	"github.com/smukkama/weather-server/pkg/config"
)

// TCPServer is the main TCP server for weather clients
type TCPServer struct {
	config       *config.TCPServerConfig
	connManager  *connection.Manager
	timerManager *timer.TimerManager
	producer     *queue.Producer
	listener     net.Listener
	wg           sync.WaitGroup
	stopCh       chan struct{}
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewTCPServer creates a new TCP server
func NewTCPServer(cfg *config.TCPServerConfig, connManager *connection.Manager, timerManager *timer.TimerManager, producer *queue.Producer) *TCPServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &TCPServer{
		config:       cfg,
		connManager:  connManager,
		timerManager: timerManager,
		producer:     producer,
		stopCh:       make(chan struct{}),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start starts the TCP server
func (s *TCPServer) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	s.listener = listener
	fmt.Printf("TCP server listening on %s\n", addr)

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// Stop stops the TCP server gracefully
func (s *TCPServer) Stop() {
	close(s.stopCh)
	s.cancel()

	if s.listener != nil {
		s.listener.Close()
	}

	s.wg.Wait()
	fmt.Println("TCP server stopped")
}

func (s *TCPServer) acceptConnections() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				fmt.Printf("Failed to accept connection: %v\n", err)
				continue
			}
		}

		// Check max connections
		if s.connManager.Count() >= s.config.MaxConnections {
			fmt.Println("Maximum connections reached, rejecting connection")
			conn.Close()
			continue
		}

		// Handle connection in a new goroutine
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Generate connection ID
	connectionID := uuid.New().String()
	fmt.Printf("New connection: %s from %s\n", connectionID, conn.RemoteAddr())

	// Set identify timeout
	conn.SetReadDeadline(time.Now().Add(s.config.IdentifyTimeout))

	// Read identification message
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read identify message: %v\n", err)
		return
	}

	// Parse identification message
	msg, err := protocol.ParseMessage([]byte(line))
	if err != nil {
		fmt.Printf("Failed to parse identify message: %v\n", err)
		s.sendError(conn, "invalid message format")
		return
	}

	identifyMsg, ok := msg.(*protocol.IdentifyMessage)
	if !ok {
		fmt.Printf("Expected identify message, got %T\n", msg)
		s.sendError(conn, "expected identify message")
		return
	}

	// Register client
	if err := s.connManager.Register(connectionID, identifyMsg.Zipcode, identifyMsg.City, conn); err != nil {
		fmt.Printf("Failed to register client: %v\n", err)
		s.sendError(conn, "failed to register")
		return
	}
	defer s.connManager.Unregister(connectionID)

	fmt.Printf("Client identified: %s (zipcode=%s, city=%s)\n", connectionID, identifyMsg.Zipcode, identifyMsg.City)

	// Send acknowledgment
	ack := protocol.NewAckMessage(protocol.AckStatusIdentified)
	if err := s.sendMessage(conn, ack); err != nil {
		fmt.Printf("Failed to send ack: %v\n", err)
		return
	}

	// Schedule inactivity timer
	s.scheduleInactivityTimer(connectionID)

	// Clear read deadline for normal operation
	conn.SetReadDeadline(time.Time{})

	// Handle messages
	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		// Read message with a reasonable timeout
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		line, err := reader.ReadString('\n')
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout, continue reading
				continue
			}
			// Connection closed or error
			fmt.Printf("Connection %s closed: %v\n", connectionID, err)
			return
		}

		// Parse message
		msg, err := protocol.ParseMessage([]byte(line))
		if err != nil {
			fmt.Printf("Failed to parse message: %v\n", err)
			continue
		}

		// Handle message
		if err := s.handleMessage(connectionID, identifyMsg.Zipcode, identifyMsg.City, msg, conn); err != nil {
			fmt.Printf("Failed to handle message: %v\n", err)
		}

		// Update activity timestamp
		s.connManager.UpdateActivity(connectionID)

		// Reschedule inactivity timer
		s.scheduleInactivityTimer(connectionID)
	}
}

func (s *TCPServer) handleMessage(connectionID, zipcode, city string, msg interface{}, conn net.Conn) error {
	switch m := msg.(type) {
	case *protocol.MetricsMessage:
		return s.handleMetrics(connectionID, zipcode, city, m)

	case *protocol.KeepaliveMessage:
		return s.handleKeepalive(conn)

	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}

func (s *TCPServer) handleMetrics(connectionID, zipcode, city string, msg *protocol.MetricsMessage) error {
	// Create internal metric message
	metricMsg := &protocol.MetricMessage{
		ConnectionID: connectionID,
		Zipcode:      zipcode,
		City:         city,
		ReceivedAt:   time.Now(),
		Data:         msg.Data,
	}

	// Encode to JSON
	data, err := protocol.EncodeMetricMessage(metricMsg)
	if err != nil {
		return fmt.Errorf("failed to encode metric: %w", err)
	}

	// Publish to Kafka (key is zipcode for partitioning)
	if err := s.producer.Publish(s.ctx, zipcode, data); err != nil {
		return fmt.Errorf("failed to publish metric: %w", err)
	}

	fmt.Printf("Received metrics from %s (zipcode=%s)\n", connectionID, zipcode)
	return nil
}

func (s *TCPServer) handleKeepalive(conn net.Conn) error {
	ack := protocol.NewAckMessage(protocol.AckStatusAlive)
	return s.sendMessage(conn, ack)
}

func (s *TCPServer) sendMessage(conn net.Conn, msg interface{}) error {
	data, err := protocol.EncodeMessage(msg)
	if err != nil {
		return err
	}

	_, err = conn.Write(append(data, '\n'))
	return err
}

func (s *TCPServer) sendError(conn net.Conn, errMsg string) {
	ack := protocol.NewAckMessage(protocol.AckStatusError)
	s.sendMessage(conn, ack)
}

func (s *TCPServer) scheduleInactivityTimer(connectionID string) {
	timerID := fmt.Sprintf("inactivity-%s", connectionID)
	expiryAt := time.Now().Add(s.config.InactivityTimeout)

	callback := func() {
		fmt.Printf("Inactivity timeout for connection %s\n", connectionID)

		// Get client info
		client, exists := s.connManager.Get(connectionID)
		if !exists {
			return
		}

		// Close connection
		client.Conn.Close()

		// Unregister will happen automatically in deferred cleanup
	}

	s.timerManager.Schedule(timerID, expiryAt, callback)
}
