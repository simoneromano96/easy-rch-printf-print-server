package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"

	nats "github.com/nats-io/nats.go"
	"github.com/spf13/viper"
)

// PrintOrder struct to describe something that must be printed.
type PrintOrder struct {
	Command string `json:"command" validate:"required"`
}

type Config struct {
	NATSUrl        string `mapstructure:"NATS_URL"`
	NATSStreamName string `mapstructure:"NATS_STREAM_NAME"`
	NATSSubject    string `mapstructure:"NATS_SUBJECT"`
	RCHPrintFUrl   string `mapstructure:"RCH_PRINTF_URL"`
}

func loadConfig() (config Config, err error) {
	// Read yaml config file
	viper.AddConfigPath("environment")
	viper.SetConfigName("development")
	viper.SetConfigType("yaml")

	// Override with ENV variables
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}

func main() {
	config, err := loadConfig()
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	fmt.Println(config)

	// Connect to NATS
	nc, _ := nats.Connect(config.NATSUrl)

	// Create JetStream Context
	js, _ := nc.JetStream(nats.PublishAsyncMaxPending(1024))

	natsChannelName := fmt.Sprintf("%s.%s", config.NATSStreamName, config.NATSSubject)

	// nats str add ORDERS --subjects "ORDERS.*" --ack --max-msgs=-1 --max-bytes=-1 --max-age=1y --storage file --retention limits --max-msg-size=-1 --discard=old
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     config.NATSStreamName,
		Subjects: []string{natsChannelName},
	})

	if err != nil {
		panic(err)
	}

	// HTTP Client
	client := &http.Client{}

	// Simple Async Ephemeral Consumer
	_, err = js.Subscribe(natsChannelName, func(m *nats.Msg) {
		// Create buffer for decoder
		binData := bytes.NewBuffer(m.Data)
		dec := gob.NewDecoder(binData)

		// Decode the value.
		var printOrder PrintOrder
		err = dec.Decode(&printOrder)
		if err != nil {
			log.Print("decode error:", err)
			return
		}

		log.Println("Received a new printOrder")

		// Build a new request
		req, err := http.NewRequest("POST", config.RCHPrintFUrl, bytes.NewBuffer([]byte(printOrder.Command)))
		if err != nil {
			log.Print(err)
			return
		}
		// Set the Header here
		req.Header.Add("Content-Type", "application/xml; charset=utf-8")

		// Send the request
		resp, err := client.Do(req)
		if err != nil {
			log.Print(err)
			return
		}

		// Log the response
		fmt.Println(resp)
	})

	if err != nil {
		panic(err)
	}

	select {}
}
