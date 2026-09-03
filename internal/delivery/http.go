package delivery

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"time"
)

// requestTimeout bounds how long a single delivery attempt may take, so a
// slow or unresponsive customer endpoint can never block a worker indefinitely.
const requestTimeout = 10 * time.Second

// maxLoggedResponseBody caps how much of a response body is kept for
// delivery history, so a huge or malicious response can't bloat the database.
const maxLoggedResponseBody = 4096

// Request describes a single webhook delivery to make.
type Request struct {
	URL     string
	EventID string
	Secret  string
	Payload []byte
}

// Result describes the outcome of a single delivery attempt.
type Result struct {
	Delivered    bool
	StatusCode   int
	ResponseBody string
	Duration     time.Duration
	Err          error
}

// Client delivers webhook payloads to customer endpoints over HTTP.
type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout:   requestTimeout,
			Transport: &http.Transport{DialContext: safeDialContext},
		},
	}
}

// Deliver sends a single signed HTTP POST of the request's payload and
// reports whether it succeeded. A 2xx response counts as delivered; anything
// else (including a network error or timeout) does not. The request is
// signed per doc section 13: Webhook-ID, Webhook-Timestamp, and a
// Webhook-Signature header the receiver can independently recompute.
func (c *Client) Deliver(ctx context.Context, dr Request) Result {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dr.URL, bytes.NewReader(dr.Payload))
	if err != nil {
		return Result{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")

	timestamp := time.Now().Unix()
	req.Header.Set("Webhook-ID", dr.EventID)
	req.Header.Set("Webhook-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Webhook-Signature", Sign(dr.Secret, timestamp, dr.Payload))

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(start)
	if err != nil {
		return Result{Err: err, Duration: duration}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxLoggedResponseBody))

	return Result{
		Delivered:    resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode:   resp.StatusCode,
		ResponseBody: string(body),
		Duration:     duration,
	}
}
