package main

import (
	"encoding/json"
	"fmt"
)

type Event struct {
	ID      string `json:"id"`
	OrderID string `json:"order_id"`
	Type    string `json:"type"`
}

type Order struct {
	ID              string `json:"id"`
	Checkout        bool   `json:"checkout"`
	Fulfilled       bool   `json:"fulfilled"`
	Receipt         bool   `json:"receipt"`
	CustomerUpdated bool   `json:"customer_updated"`
}

// project deduplicates deliveries before applying state transitions.
func project(events []Event) ([]Order, error) {
	seen := map[string]bool{}
	orders := map[string]*Order{}
	sequence := []string{}
	for _, event := range events {
		if event.ID == "" || event.OrderID == "" {
			return nil, fmt.Errorf("event id and order_id are required")
		}
		if seen[event.ID] {
			continue
		}
		seen[event.ID] = true
		order := orders[event.OrderID]
		if order == nil {
			order = &Order{ID: event.OrderID}
			orders[event.OrderID] = order
			sequence = append(sequence, event.OrderID)
		}
		switch event.Type {
		case "checkout":
			order.Checkout = true
		case "fulfillment":
			order.Fulfilled = true
		case "receipt":
			order.Receipt = true
		case "customer_order_update":
			order.CustomerUpdated = true
		default:
			return nil, fmt.Errorf("unknown event type %q", event.Type)
		}
	}
	out := make([]Order, 0, len(sequence))
	for _, id := range sequence {
		out = append(out, *orders[id])
	}
	return out, nil
}

func decodeEvents(data json.RawMessage) ([]Event, error) {
	var deliveries struct {
		Items []Event `json:"items"`
	}
	if err := json.Unmarshal(data, &deliveries); err == nil {
		return deliveries.Items, nil
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("decode delivery records: %w", err)
	}
	return events, nil
}
