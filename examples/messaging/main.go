// Example: Kafka producer + RabbitMQ publisher with request-ID propagation.
//
// Prerequisites: run `make docker-up` first (starts Kafka + RabbitMQ containers).
//
// Run:
//
//	cd examples/messaging && go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rs/zerolog"

	applogger "github.com/abdullahPrasetio/wapgo/pkg/logger"
	"github.com/abdullahPrasetio/wapgo/pkg/messaging/kafka"
	"github.com/abdullahPrasetio/wapgo/pkg/messaging/rabbitmq"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	ctx := applogger.WithRequestID(context.Background(), "example-request-id-001")

	// ── Kafka ─────────────────────────────────────────────────────────────────
	fmt.Println("=== Kafka Producer ===")
	brokers := "localhost:9092"

	status := kafka.HealthCheck(brokers)(ctx)
	fmt.Printf("Kafka health: %s\n", status)

	if status == "ok" {
		producer := kafka.NewProducer(brokers, logger)
		defer producer.Close()

		for i := range 3 {
			msg := kafka.Message{
				Topic: "user-events",
				Key:   []byte(fmt.Sprintf("key-%d", i+1)),
				Value: []byte(fmt.Sprintf(`{"event":"user.created","id":%d,"ts":"%s"}`, i+1, time.Now().Format(time.RFC3339))),
			}
			if err := producer.Publish(ctx, msg); err != nil {
				log.Printf("kafka publish: %v", err)
			} else {
				fmt.Printf("  published msg %d to %s\n", i+1, msg.Topic)
			}
		}
	}

	// ── RabbitMQ ──────────────────────────────────────────────────────────────
	fmt.Println("\n=== RabbitMQ Publisher ===")
	dsn := "amqp://guest:guest@localhost:5672/"

	rmqStatus := rabbitmq.HealthCheck(dsn)(ctx)
	fmt.Printf("RabbitMQ health: %s\n", rmqStatus)

	if rmqStatus == "ok" {
		publisher, err := rabbitmq.NewPublisher(dsn, "user-events", logger)
		if err != nil {
			log.Fatalf("rabbitmq publisher: %v", err)
		}
		defer publisher.Close()

		for i := range 3 {
			msg := rabbitmq.Message{
				RoutingKey: "user.created",
				Body:       []byte(fmt.Sprintf(`{"event":"user.created","id":%d}`, i+1)),
			}
			if err := publisher.Publish(ctx, msg); err != nil {
				log.Printf("rabbitmq publish: %v", err)
			} else {
				fmt.Printf("  published msg %d\n", i+1)
			}
		}

		// ── Consumer ─────────────────────────────────────────────────────────
		fmt.Println("\n=== RabbitMQ Consumer (drain 3 messages, 3s timeout) ===")
		consumer, err := rabbitmq.NewConsumer(dsn, "user-events", logger)
		if err != nil {
			log.Fatalf("rabbitmq consumer: %v", err)
		}
		defer consumer.Close()

		received := make(chan struct{}, 3)
		if err := consumer.Subscribe("user-q", "user.created", func(_ context.Context, msg rabbitmq.Message) error {
			fmt.Printf("  received: key=%s body=%s\n", msg.RoutingKey, msg.Body)
			received <- struct{}{}
			return nil
		}); err != nil {
			log.Fatalf("subscribe: %v", err)
		}

		timeout := time.After(3 * time.Second)
		for range 3 {
			select {
			case <-received:
			case <-timeout:
				fmt.Println("  timeout waiting for messages")
				return
			}
		}
		fmt.Println("  all 3 messages received")
	}
}
