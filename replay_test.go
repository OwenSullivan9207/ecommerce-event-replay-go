package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDecodeEvents(t *testing.T) {
	want := []Event{{ID: "evt-1", OrderID: "order-1", Type: "checkout"}}
	for _, input := range []string{
		`{"items":[{"id":"evt-1","order_id":"order-1","type":"checkout"}]}`,
		`[{"id":"evt-1","order_id":"order-1","type":"checkout"}]`,
	} {
		got, err := decodeEvents(json.RawMessage(input))
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("decodeEvents(%s) = %v, %v; want %v", input, got, err, want)
		}
	}
}

func TestProject(t *testing.T) {
	tests := []struct {
		name   string
		events []Event
		want   []Order
		bad    bool
	}{
		{"duplicate receipt", []Event{{"1", "o-7", "checkout"}, {"2", "o-7", "receipt"}, {"2", "o-7", "receipt"}, {"3", "o-7", "fulfillment"}, {"4", "o-7", "customer_order_update"}}, []Order{{"o-7", true, true, true, true}}, false},
		{"unknown transition", []Event{{"1", "o-7", "refund"}}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := project(tt.events)
			if (err != nil) != tt.bad || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("project() = %v, %v; want %v", got, err, tt.want)
			}
		})
	}
}
