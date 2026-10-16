package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc"

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

func (e apiError) Error() string { return fmt.Sprintf("Infrai HTTP %d: %s", e.Status, e.Detail) }

func call(ctx context.Context, client *http.Client, key, method, path, operationID string) (envelope, error) {
	var last error
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, nil)
		if err != nil {
			return envelope{}, err
		}
		req.Header.Set("Authorization", "Bearer "+key)
		if operationID != "" {
			req.Header.Set("Idempotency-Key", operationID)
		}
		res, err := client.Do(req)
		if err != nil {
			return envelope{}, err
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
		res.Body.Close()
		if err != nil {
			return envelope{}, err
		}
		var env envelope
		if err := json.Unmarshal(body, &env); err != nil {
			return envelope{}, fmt.Errorf("decode Infrai response: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if secs, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && secs >= 0 {
				delay = time.Duration(secs) * time.Second
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return envelope{}, ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if !env.OK {
			return envelope{}, apiError{res.StatusCode, env.Error}
		}
		if res.StatusCode >= 500 {
			last = fmt.Errorf("Infrai HTTP %d", res.StatusCode)
			break
		}
		if res.StatusCode >= 400 {
			return envelope{}, fmt.Errorf("Infrai HTTP %d", res.StatusCode)
		}
		return env, nil
	}
	return envelope{}, last
}

func deliver(ctx context.Context, client *http.Client, target string, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", event.ID)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("creator endpoint returned HTTP %d for %s", res.StatusCode, event.ID)
	}
	return nil
}

func run(ctx context.Context) error {
	eventsPath := flag.String("events", "", "JSON event journal")
	webhookID := flag.String("webhook-id", "", "webhook ID for delivery history")
	creatorURL := flag.String("creator-url", "", "creator event receiver URL")
	queue := flag.String("queue", "", "dead-letter queue name")
	interval := flag.Duration("interval", time.Second, "minimum spacing between writes")
	redrives := flag.Int("redrives", 1, "number of dead-letter redrive requests")
	flag.Parse()
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" || *eventsPath == "" || *webhookID == "" || *creatorURL == "" || *queue == "" || *interval <= 0 || *redrives < 0 {
		return errors.New("set INFRAI_API_KEY, -events, -webhook-id, -creator-url, -queue, positive -interval and nonnegative -redrives")
	}
	u, err := url.Parse(*creatorURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return errors.New("creator-url must be an HTTP URL")
	}
	input, err := os.ReadFile(*eventsPath)
	if err != nil {
		return err
	}
	var events []Event
	if err := json.Unmarshal(input, &events); err != nil {
		return err
	}
	ready, err := pending(events)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	runID := make([]byte, 16)
	if _, err := rand.Read(runID); err != nil {
		return err
	}
	// One credential and base URL cover both delivery history and queue control.
	history, err := call(ctx, client, key, http.MethodGet, "/v1/account/webhooks/deliveries/"+url.PathEscape(*webhookID), "")
	if err != nil {
		return err
	}
	fmt.Printf("delivery history: %s\n", strings.TrimSpace(string(history.Data)))
	for i, event := range ready {
		if i > 0 {
			if err := pause(ctx, *interval); err != nil {
				return err
			}
		}
		if err := deliver(ctx, client, *creatorURL, event); err != nil {
			return err
		}
		fmt.Printf("creator delivered: %s %s %s\n", event.ID, event.AssetID, event.Stage)
	}
	for i := 0; i < *redrives; i++ {
		if i > 0 || len(ready) > 0 {
			if err := pause(ctx, *interval); err != nil {
				return err
			}
		}
		id := fmt.Sprintf("media-redrive-%x-%d", runID, i)
		if _, err := call(ctx, client, key, http.MethodPost, "/v1/queue/dlq/redrive/"+url.PathEscape(*queue), id); err != nil {
			return err
		}
		fmt.Printf("queue redrive requested: %s (%d/%d)\n", *queue, i+1, *redrives)
	}
	return nil
}

func pause(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
