package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"
)

// Sample weather client that simulates a weather station

type IdentifyMessage struct {
	Type    string `json:"type"`
	Zipcode string `json:"zipcode"`
	City    string `json:"city"`
}

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

type MetricsMessage struct {
	Type string     `json:"type"`
	Data MetricData `json:"data"`
}

type KeepaliveMessage struct {
	Type string `json:"type"`
}

type AckMessage struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

func main() {
	// Configuration
	serverAddr := "localhost:8080"
	zipcode := "90210"
	city := "Beverly Hills"
	metricsInterval := 30 * time.Second // Reduced for demo (normally 5 minutes)
	keepaliveInterval := 15 * time.Second

	fmt.Printf("Weather Client Starting...\n")
	fmt.Printf("Location: %s, %s\n", city, zipcode)
	fmt.Printf("Server: %s\n\n", serverAddr)

	// Connect to server
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	fmt.Println("✓ Connected to server")

	reader := bufio.NewReader(conn)

	// Send identify message
	identify := IdentifyMessage{
		Type:    "identify",
		Zipcode: zipcode,
		City:    city,
	}
	if err := sendMessage(conn, identify); err != nil {
		log.Fatalf("Failed to send identify: %v", err)
	}
	fmt.Println("→ Sent identify message")

	// Wait for acknowledgment
	ack, err := readAck(reader)
	if err != nil {
		log.Fatalf("Failed to read ack: %v", err)
	}
	fmt.Printf("← Received ack: %s\n\n", ack.Status)

	// Start goroutines for periodic tasks
	metricsTicker := time.NewTicker(metricsInterval)
	keepaliveTicker := time.NewTicker(keepaliveInterval)
	defer metricsTicker.Stop()
	defer keepaliveTicker.Stop()

	// Background goroutine to read server responses
	go func() {
		for {
			ack, err := readAck(reader)
			if err != nil {
				fmt.Printf("Connection closed: %v\n", err)
				return
			}
			fmt.Printf("← Received ack: %s\n", ack.Status)
		}
	}()

	fmt.Println("✓ Client running (Ctrl+C to stop)")

	// Send initial metrics
	sendWeatherMetrics(conn)

	// Main loop
	for {
		select {
		case <-metricsTicker.C:
			sendWeatherMetrics(conn)

		case <-keepaliveTicker.C:
			sendKeepalive(conn)
		}
	}
}

func sendWeatherMetrics(conn net.Conn) {
	// Generate realistic-ish random weather data
	temp := 15.0 + rand.Float64()*20.0     // 15-35°C
	humidity := 30.0 + rand.Float64()*50.0 // 30-80%
	precip := 0.0
	if rand.Float64() < 0.2 { // 20% chance of rain
		precip = rand.Float64() * 10.0
	}
	windSpeed := rand.Float64() * 30.0 // 0-30 mph
	directions := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	windDir := directions[rand.Intn(len(directions))]
	pollution := 20.0 + rand.Float64()*80.0 // 20-100
	pollen := 10.0 + rand.Float64()*90.0    // 10-100

	metrics := MetricsMessage{
		Type: "metrics",
		Data: MetricData{
			Timestamp:      time.Now().UTC().Format(time.RFC3339),
			Temperature:    roundFloat(temp, 2),
			Humidity:       roundFloat(humidity, 2),
			Precipitation:  roundFloat(precip, 2),
			WindSpeed:      roundFloat(windSpeed, 2),
			WindDirection:  windDir,
			PollutionIndex: roundFloat(pollution, 2),
			PollenIndex:    roundFloat(pollen, 2),
		},
	}

	if err := sendMessage(conn, metrics); err != nil {
		log.Printf("Failed to send metrics: %v", err)
		return
	}

	fmt.Printf("→ Sent metrics: temp=%.1f°C, humidity=%.1f%%, wind=%.1f mph %s\n",
		temp, humidity, windSpeed, windDir)
}

func sendKeepalive(conn net.Conn) {
	keepalive := KeepaliveMessage{Type: "keepalive"}
	if err := sendMessage(conn, keepalive); err != nil {
		log.Printf("Failed to send keepalive: %v", err)
		return
	}
	fmt.Println("→ Sent keepalive")
}

func sendMessage(conn net.Conn, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(data, '\n'))
	return err
}

func readAck(reader *bufio.Reader) (*AckMessage, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	var ack AckMessage
	if err := json.Unmarshal([]byte(line), &ack); err != nil {
		return nil, err
	}

	return &ack, nil
}

func roundFloat(val float64, precision int) float64 {
	ratio := 1.0
	for i := 0; i < precision; i++ {
		ratio *= 10
	}
	return float64(int(val*ratio)) / ratio
}
