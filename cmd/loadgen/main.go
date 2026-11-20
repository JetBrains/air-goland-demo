// Command loadgen sends a steady stream of signed webhook events to a running
// service, so there's real traffic to profile.
//
// Start the service first, then run:
//
//	go run ./cmd/loadgen
package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	url := flag.String("url", "http://localhost:8080/webhooks/stripe", "webhook endpoint to send events to")
	duration := flag.Duration("duration", 30*time.Second, "how long to send events for")
	workers := flag.Int("workers", 8, "number of concurrent senders")
	flag.Parse()

	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}

	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{MaxIdleConnsPerHost: *workers},
	}

	// Every event gets a new ID, so the service validates and stores each one
	// instead of answering duplicates. The run prefix keeps IDs unique across runs.
	run := time.Now().Unix() % 1e11
	var next atomic.Int64

	var mu sync.Mutex
	statuses := map[int]int{}
	failed := 0

	deadline := time.Now().Add(*duration)
	log.Printf("sending events to %s for %s with %d workers", *url, *duration, *workers)

	var wg sync.WaitGroup
	for range *workers {
		wg.Go(func() {
			for time.Now().Before(deadline) {
				id := fmt.Sprintf("evt_%011d%013d", run, next.Add(1))
				status, err := send(client, *url, []byte(secret), id)

				mu.Lock()
				if err != nil {
					failed++
				} else {
					statuses[status]++
				}
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	total := failed
	for _, n := range statuses {
		total += n
	}
	log.Printf("sent %d events (%.0f per second)", total, float64(total)/duration.Seconds())
	for status, n := range statuses {
		log.Printf("  %d %s: %d", status, http.StatusText(status), n)
	}
	if failed > 0 {
		log.Printf("  failed to send: %d (is the service running?)", failed)
	}
}

func send(client *http.Client, url string, secret []byte, id string) (int, error) {
	body := fmt.Appendf(nil,
		`{"id": %q, "type": "payment.succeeded", "data": {"amount": 2500, "currency": "gbp"}}`, id)

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(ts + "."))
	mac.Write(body)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Webhook-Signature", "t="+ts+",v1="+hex.EncodeToString(mac.Sum(nil)))

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}
