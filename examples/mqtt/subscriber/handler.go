package main

import (
	"encoding/json"
	"fmt"

	"github.com/eclipse/paho.golang/paho"
)

type handler struct{}

func NewHandler() *handler {
	return &handler{}
}

type Message struct {
	Count uint64
}

func (o *handler) handle(msg *paho.Publish) {
	// We extract the count and write that out first to simplify checking for missing values
	var m Message
	if err := json.Unmarshal(msg.Payload, &m); err != nil {
		fmt.Printf("Message could not be parsed (%s): %s", msg.Payload, err)
	}

	fmt.Printf("received message: %s\n", msg.Payload)
}
