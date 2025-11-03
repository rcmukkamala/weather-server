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

// ConnectionJob represents a job to process data from a connection
type ConnectionJob struct {
	ConnectionID string
	Zipcode      string
	City         string
	Data         []byte
	Conn         net.Conn
	Timestamp    time.Time
}

// WorkerPoolTCPServer is a TCP server using worker pool pattern
type WorkerPoolTCPServer struct {
	config       *config.TCPServerConfig
	connManager  *connection.Manager
	timerManager *timer.TimerManager
	producer     *queue.Producer
	listener     net.Listener

	// Worker pool components
	jobQueue    chan *ConnectionJob
	workerCount int
	workers     []*Worker

	wg     sync.WaitGroup
	stopCh chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
}

// Worker represents a worker that processes connection jobs
type Worker struct {
	id       int
	jobQueue <-chan *ConnectionJob
	server   *WorkerPoolTCPServer
	stopCh   <-chan struct{}
}

// NewWorkerPoolTCPServer creates a new worker pool TCP server
func NewWorkerPoolTCPServer(
	cfg *config.TCPServerConfig,
	connManager *connection.Manager,
	timerManager *timer.TimerManager,
	producer *queue.Producer,
	workerCount int,
	jobQueueSize int,
) *WorkerPoolTCPServer {
	ctx, cancel := context.WithCancel(context.Background())

	if workerCount <= 0 {
		workerCount = 10 // Default 10 workers
	}

	if jobQueueSize <= 0 {
		jobQueueSize = 1000 // Default queue size
	}

	return &WorkerPoolTCPServer{
		config:       cfg,
		connManager:  connManager,
		timerManager: timerManager,
		producer:     producer,
		jobQueue:     make(chan *ConnectionJob, jobQueueSize),
		workerCount:  workerCount,
		stopCh:       make(chan struct{}),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start starts the TCP server and worker pool
func (s *WorkerPoolTCPServer) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	s.listener = listener
	fmt.Printf("Worker Pool TCP server listening on %s with %d workers\n", addr, s.workerCount)

	// Start workers
	s.startWorkers()

	// Start accepting connections
	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// Stop stops the TCP server gracefully
func (s *WorkerPoolTCPServer) Stop() {
	fmt.Println("Stopping Worker Pool TCP server...")
	close(s.stopCh)
	s.cancel()

	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for accept loop to finish
	s.wg.Wait()

	// Close job queue (no more jobs)
	close(s.jobQueue)

	// Workers will exit when jobQueue is closed
	fmt.Println("Worker Pool TCP server stopped")
}

// startWorkers initializes and starts worker goroutines
func (s *WorkerPoolTCPServer) startWorkers() {
	s.workers = make([]*Worker, s.workerCount)

	for i := 0; i < s.workerCount; i++ {
		worker := &Worker{
			id:       i,
			jobQueue: s.jobQueue,
			server:   s,
			stopCh:   s.stopCh,
		}
		s.workers[i] = worker

		s.wg.Add(1)
		go worker.Start(&s.wg)
	}

	fmt.Printf("Started %d workers\n", s.workerCount)
}

// acceptConnections accepts incoming connections
func (s *WorkerPoolTCPServer) acceptConnections() {
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

		// Handle connection in a lightweight goroutine (just for reading)
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles initial handshake and reads from connection
// This goroutine is lightweight - it only reads and dispatches to workers
func (s *WorkerPoolTCPServer) handleConnection(conn net.Conn) {
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

	// Read messages and dispatch to workers
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

		// Create job and send to worker pool
		job := &ConnectionJob{
			ConnectionID: connectionID,
			Zipcode:      identifyMsg.Zipcode,
			City:         identifyMsg.City,
			Data:         []byte(line),
			Conn:         conn,
			Timestamp:    time.Now(),
		}

		// Non-blocking send to job queue
		select {
		case s.jobQueue <- job:
			// Job queued successfully
		case <-s.stopCh:
			return
		default:
			// Queue is full, log and drop (or implement backpressure)
			fmt.Printf("Job queue full, dropping message from %s\n", connectionID)
		}

		// Update activity timestamp
		s.connManager.UpdateActivity(connectionID)

		// Reschedule inactivity timer
		s.scheduleInactivityTimer(connectionID)
	}
}

// Worker methods

// Start starts the worker
func (w *Worker) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Printf("Worker %d started\n", w.id)

	for {
		select {
		case job, ok := <-w.jobQueue:
			if !ok {
				// Channel closed, worker should exit
				fmt.Printf("Worker %d stopped\n", w.id)
				return
			}
			w.processJob(job)

		case <-w.stopCh:
			fmt.Printf("Worker %d received stop signal\n", w.id)
			return
		}
	}
}

// processJob processes a connection job
func (w *Worker) processJob(job *ConnectionJob) {
	// Parse message
	msg, err := protocol.ParseMessage(job.Data)
	if err != nil {
		fmt.Printf("Worker %d: Failed to parse message: %v\n", w.id, err)
		return
	}

	// Handle message based on type
	switch m := msg.(type) {
	case *protocol.MetricsMessage:
		if err := w.handleMetrics(job, m); err != nil {
			fmt.Printf("Worker %d: Failed to handle metrics: %v\n", w.id, err)
		}

	case *protocol.KeepaliveMessage:
		if err := w.handleKeepalive(job); err != nil {
			fmt.Printf("Worker %d: Failed to handle keepalive: %v\n", w.id, err)
		}

	default:
		fmt.Printf("Worker %d: Unknown message type: %T\n", w.id, msg)
	}
}

// handleMetrics handles metrics message
func (w *Worker) handleMetrics(job *ConnectionJob, msg *protocol.MetricsMessage) error {
	// Create internal metric message
	metricMsg := &protocol.MetricMessage{
		ConnectionID: job.ConnectionID,
		Zipcode:      job.Zipcode,
		City:         job.City,
		ReceivedAt:   job.Timestamp,
		Data:         msg.Data,
	}

	// Encode to JSON
	data, err := protocol.EncodeMetricMessage(metricMsg)
	if err != nil {
		return fmt.Errorf("failed to encode metric: %w", err)
	}

	// Publish to Kafka (key is zipcode for partitioning)
	if err := w.server.producer.Publish(w.server.ctx, job.Zipcode, data); err != nil {
		return fmt.Errorf("failed to publish metric: %w", err)
	}

	fmt.Printf("Worker %d: Received metrics from %s (zipcode=%s)\n", w.id, job.ConnectionID, job.Zipcode)
	return nil
}

// handleKeepalive handles keepalive message
func (w *Worker) handleKeepalive(job *ConnectionJob) error {
	ack := protocol.NewAckMessage(protocol.AckStatusAlive)
	return w.server.sendMessage(job.Conn, ack)
}

// Helper methods

func (s *WorkerPoolTCPServer) sendMessage(conn net.Conn, msg interface{}) error {
	data, err := protocol.EncodeMessage(msg)
	if err != nil {
		return err
	}

	_, err = conn.Write(append(data, '\n'))
	return err
}

func (s *WorkerPoolTCPServer) sendError(conn net.Conn, errMsg string) {
	ack := protocol.NewAckMessage(protocol.AckStatusError)
	s.sendMessage(conn, ack)
}

func (s *WorkerPoolTCPServer) scheduleInactivityTimer(connectionID string) {
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
