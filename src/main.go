package main

import (
	"fmt"
	"time"

	nats "github.com/nats-io/nats.go"
)

func main() {
	// Connect to NATS
	nc, _ := nats.Connect(nats.DefaultURL)

	// Create JetStream Context
	js, _ := nc.JetStream(nats.PublishAsyncMaxPending(1024))

	// nats str add ORDERS --subjects "ORDERS.*" --ack --max-msgs=-1 --max-bytes=-1 --max-age=1y --storage file --retention limits --max-msg-size=-1 --discard=old
	si, err := js.AddStream(&nats.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"ORDERS.*"},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(si)

	// Simple Async Stream Publisher
	for i := 0; i < 500; i++ {
		js.PublishAsync("ORDERS.scratch", []byte(fmt.Sprintf("order %d", i)))
	}
	select {
	case <-js.PublishAsyncComplete():
		fmt.Println("Resolved publish")
	case <-time.After(5 * time.Second):
		fmt.Println("Did not resolve in time")
	}

	// Simple Async Ephemeral Consumer
	sub, err := js.Subscribe("ORDERS.*", func(m *nats.Msg) {
		fmt.Printf("Received a JetStream message: %s\n", string(m.Data))
	})

	if err != nil {
		panic(err)
	}

	// Unsubscribe
	sub.Unsubscribe()

	// Drain
	sub.Drain()
}
