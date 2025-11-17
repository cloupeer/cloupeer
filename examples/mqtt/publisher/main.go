package main

import (
	"flag"
	"log"
)

func main() {
	// Define flags
	debug := flag.Bool("debug", false, "debug log")
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	topic := flag.String("topic", "test/topic/go", "MQTT topic to subscribe to")
	clientID := flag.String("clientid", "publisher-001", "MQTT Client ID")
	qos := flag.Int("qos", 1, "MQTT QoS level")
	flag.Parse()

	// Create config
	cfg := Config{
		Debug:    *debug,
		Broker:   *broker,
		Topic:    *topic,
		ClientID: *clientID,
		QoS:      *qos,
	}

	// Create and run the application
	app := NewPublisher(cfg)
	if err := app.Run(); err != nil {
		log.Fatalf("Application run failed: %v", err)
	}
}
