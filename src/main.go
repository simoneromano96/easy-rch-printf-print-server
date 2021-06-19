package main

import (
	"bytes"
	"encoding/gob"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"

	nats "github.com/nats-io/nats.go"
	"github.com/spf13/viper"
)

// RCH Request to be sent
type Service struct {
	Commands []string `xml:"cmd"`
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
		log.Println(string(m.Data))

		dec := gob.NewDecoder(binData)

		// Decode the value.
		var rchRequest Service
		err = dec.Decode(&rchRequest)
		if err != nil {
			log.Print("decode error:", err)
			return
		}

		log.Println("Received a new request")
		log.Println(rchRequest)

		// Initialize the encoder
		var buffer bytes.Buffer
		// Write to buffer
		enc := xml.NewEncoder(&buffer)
		err := enc.Encode(rchRequest)
		if err != nil {
			log.Print(err)
			return
		}

		log.Println("Encoded request")
		log.Println(buffer.String())

		// Build a new request
		req, err := http.NewRequest("POST", config.RCHPrintFUrl, bytes.NewBuffer(buffer.Bytes()))
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
