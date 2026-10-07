package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type replayRequest struct {
	WebhookID       string `json:"webhook_id"`
	Queue           string `json:"queue"`
	ReplayID        string `json:"replay_id"`
	IntervalSeconds int    `json:"interval_seconds"`
	Batches         int    `json:"batches"`
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	client := apiClient{key: key, http: &http.Client{Timeout: 30 * time.Second}}
	http.HandleFunc("POST /replay", func(w http.ResponseWriter, r *http.Request) {
		var input replayRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.WebhookID == "" || input.Queue == "" || input.ReplayID == "" || input.Batches < 1 || input.Batches > 20 || input.IntervalSeconds < 1 {
			http.Error(w, "webhook_id, queue, replay_id, batches (1-20), interval_seconds (>=1) required", http.StatusBadRequest)
			return
		}
		data, err := client.deliveries(r.Context(), input.WebhookID)
		if err != nil {
			respondError(w, err)
			return
		}
		events, err := decodeEvents(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		orders, err := project(events)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		for i := 0; i < input.Batches; i++ {
			if i > 0 {
				select {
				case <-time.After(time.Duration(input.IntervalSeconds) * time.Second):
				case <-r.Context().Done():
					return
				}
			}
			if err := client.redrive(r.Context(), input.Queue, fmt.Sprintf("%s-%d", input.ReplayID, i)); err != nil {
				respondError(w, err)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(struct {
			Orders   []Order `json:"orders"`
			Redriven int     `json:"redriven"`
		}{orders, input.Batches}); err != nil {
			log.Print(err)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func respondError(w http.ResponseWriter, err error) {
	var remote apiError
	if errors.As(err, &remote) && remote.Status >= 400 && remote.Status < 500 {
		http.Error(w, remote.Error(), remote.Status)
		return
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	log.Print(fmt.Errorf("replay: %w", err))
	http.Error(w, "upstream request failed", http.StatusBadGateway)
}
