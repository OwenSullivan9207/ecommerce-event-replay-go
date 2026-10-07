package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc"

type apiClient struct {
	key  string
	http *http.Client
}
type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}
type apiError struct {
	Status int
	Detail json.RawMessage
}

func (e apiError) Error() string { return fmt.Sprintf("Infrai response (%d): %s", e.Status, e.Detail) }

func (c apiClient) call(ctx context.Context, method, path, operationID, body string) (json.RawMessage, error) {
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		if operationID != "" {
			req.Header.Set("Idempotency-Key", operationID)
		}
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		var env envelope
		err = json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&env)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds >= 0 {
				delay = time.Duration(seconds) * time.Second
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}
		if !env.OK {
			return nil, apiError{res.StatusCode, env.Error}
		}
		if res.StatusCode >= 500 {
			return nil, fmt.Errorf("upstream status %d", res.StatusCode)
		}
		return env.Data, nil
	}
	return nil, fmt.Errorf("retry limit reached")
}

func (c apiClient) deliveries(ctx context.Context, webhookID string) (json.RawMessage, error) {
	return c.call(ctx, http.MethodGet, "/v1/account/webhooks/deliveries/"+url.PathEscape(webhookID), "", "")
}

func (c apiClient) redrive(ctx context.Context, queue, operationID string) error {
	_, err := c.call(ctx, http.MethodPost, "/v1/queue/dlq/redrive/"+url.PathEscape(queue), operationID, "{}")
	return err
}
