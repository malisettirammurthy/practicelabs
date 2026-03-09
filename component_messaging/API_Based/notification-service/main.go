package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type NotificationRequest struct {
	EventType  string `json:"event_type"`
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	Message    string `json:"message"`
}

type NotificationRecord struct {
	EventType  string    `json:"event_type"`
	OrderID    string    `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	Message    string    `json:"message"`
	ReceivedAt time.Time `json:"received_at"`
}

var (
	notifications []NotificationRecord
	mu            sync.Mutex
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"service":"notification-service","status":"up"}`))
}

func notifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.OrderID == "" || req.CustomerID == "" {
		http.Error(w, `{"error":"order_id and customer_id are required"}`, http.StatusBadRequest)
		return
	}

	record := NotificationRecord{
		EventType:  req.EventType,
		OrderID:    req.OrderID,
		CustomerID: req.CustomerID,
		Message:    req.Message,
		ReceivedAt: time.Now().UTC(),
	}

	mu.Lock()
	notifications = append(notifications, record)
	mu.Unlock()

	log.Printf("Notification received: order_id=%s customer_id=%s message=%s",
		req.OrderID, req.CustomerID, req.Message)

	resp := map[string]interface{}{
		"message": "notification processed",
		"record":  record,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

func listNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	resp := map[string]interface{}{
		"count":         len(notifications),
		"notifications": notifications,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/notify", notifyHandler)
	http.HandleFunc("/notifications", listNotificationsHandler)

	port := 8081
	fmt.Printf("notification-service running on port %d\n", port)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
