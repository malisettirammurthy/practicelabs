package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationEvent struct {
	EventType string                 `json:"event_type"`
	Target    string                 `json:"target"`
	Order     map[string]interface{} `json:"order"`
	Message   string                 `json:"message"`
}

type NotificationRecord struct {
	EventType  string                 `json:"event_type"`
	Message    string                 `json:"message"`
	Order      map[string]interface{} `json:"order"`
	ReceivedAt time.Time              `json:"received_at"`
}

var (
	notifications []NotificationRecord
	mu            sync.Mutex
)

const (
	exchangeName = "order.events"
	queueName    = "notification.queue"
	routingKey   = "order.created.notification"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"service":"notification-service","status":"up"}`))
}

func listNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	resp := map[string]interface{}{
		"count":         len(notifications),
		"notifications": notifications,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func consumeRabbitMQ() {
	rabbitHost := os.Getenv("RABBITMQ_HOST")
	if rabbitHost == "" {
		rabbitHost = "localhost"
	}

	rabbitUser := os.Getenv("RABBITMQ_USERNAME")
	if rabbitUser == "" {
		rabbitUser = "guest"
	}

	rabbitPass := os.Getenv("RABBITMQ_PASSWORD")
	if rabbitPass == "" {
		rabbitPass = "guest"
	}

	connStr := fmt.Sprintf("amqp://%s:%s@%s:5672/", rabbitUser, rabbitPass, rabbitHost)

	for {
		conn, err := amqp.Dial(connStr)
		if err != nil {
			log.Printf("RabbitMQ connection failed: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		ch, err := conn.Channel()
		if err != nil {
			log.Printf("Channel creation failed: %v", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		_, err = ch.ExchangeDeclare(
			exchangeName,
			"direct",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Printf("Exchange declare failed: %v", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		q, err := ch.QueueDeclare(
			queueName,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Printf("Queue declare failed: %v", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		err = ch.QueueBind(
			q.Name,
			routingKey,
			exchangeName,
			false,
			nil,
		)
		if err != nil {
			log.Printf("Queue bind failed: %v", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		msgs, err := ch.Consume(
			q.Name,
			"",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Printf("Consume failed: %v", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("Waiting for messages on %s", queueName)

		for msg := range msgs {
			var event NotificationEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("Invalid message: %v", err)
				msg.Nack(false, false)
				continue
			}

			record := NotificationRecord{
				EventType:  event.EventType,
				Message:    event.Message,
				Order:      event.Order,
				ReceivedAt: time.Now().UTC(),
			}

			mu.Lock()
			notifications = append(notifications, record)
			mu.Unlock()

			log.Printf("Notification consumed: %+v", record)
			msg.Ack(false)
		}

		ch.Close()
		conn.Close()
		time.Sleep(5 * time.Second)
	}
}

func main() {
	go consumeRabbitMQ()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/notifications", listNotificationsHandler)

	log.Println("notification-service listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
