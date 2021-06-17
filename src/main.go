package main

import (
	"fmt"
	"log"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/spf13/viper"
)

type Config struct {
	NATSUrl        string `mapstructure:"NATS_URL"`
	NATSStreamName string `mapstructure:"NATS_STREAM_NAME"`
	NATSSubject    string `mapstructure:"NATS_SUBJECT"`
	RCHPrintFUrl   string `mapstructure:"RCH_PRINTF_URL"`
}

func loadConfig() (config Config, err error) {
	viper.AddConfigPath("environment")
	viper.SetConfigName("development")
	viper.SetConfigType("yaml")

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

	/*
		fmt.Println(os.Getenv("NATS_URL"))
		fmt.Println(os.Getenv("NATS_STREAM_NAME"))
		fmt.Println(os.Getenv("NATS_SUBJECT"))
		fmt.Println(os.Getenv("RCH_PRINTF_URL"))
	*/

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
	// fmt.Println(si)

	// Simple Async Stream Publisher
	for i := 0; i < 500; i++ {
		js.PublishAsync(natsChannelName, []byte(fmt.Sprintf("order %d", i)))
	}
	select {
	case <-js.PublishAsyncComplete():
		fmt.Println("Resolved publish")
	case <-time.After(5 * time.Second):
		fmt.Println("Did not resolve in time")
	}

	time.Sleep(20 * time.Second)

	// Simple Async Ephemeral Consumer
	sub, err := js.Subscribe(natsChannelName, func(m *nats.Msg) {
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
