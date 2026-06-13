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
		// One shared connection for publisher + consumer in this process.
		rmqConn, err := rabbitmq.NewConnection(dsn, logger)
		if err != nil {
			log.Fatalf("rabbitmq connection: %v", err)
		}
		defer rmqConn.Close() //nolint:errcheck

		publisher, err := rabbitmq.NewPublisher(rmqConn, "user-events", logger)
		if err != nil {
			log.Fatalf("rabbitmq publisher: %v", err)
		}
		defer publisher.Close() //nolint:errcheck

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

		fmt.Println("\n=== RabbitMQ Consumer (drain 3 messages, 3s timeout) ===")
		consumer := rabbitmq.NewConsumer(rmqConn, "user-events", logger)

		received := make(chan struct{}, 3)
		subCtx, subCancel := context.WithTimeout(ctx, 3*time.Second)
		defer subCancel()
		go func() {
			consumer.Subscribe(subCtx, "user-q", "user.created", func(_ context.Context, msg rabbitmq.Message) error { //nolint:errcheck
				fmt.Printf("  received: key=%s body=%s\n", msg.RoutingKey, msg.Body)
				received <- struct{}{}
				return nil
			})
		}()

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
