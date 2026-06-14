// Package zenquotes is the library behind the zenquotes command line:
// the HTTP client, request shaping, and the typed data models for the
// ZenQuotes.io public motivational quote API.
//
// The API requires no authentication. A polite User-Agent and 500 ms pacing
// between requests keeps the client well within the free-tier rate limits.
package zenquotes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Host is the site this client talks to.
const Host = "zenquotes.io"

// Config holds all tunable parameters for the Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://zenquotes.io",
		UserAgent: "zenquotes-cli/0.1 (tamnd87@gmail.com)",
		Rate:      500 * time.Millisecond,
		Timeout:   10 * time.Second,
		Retries:   3,
	}
}

// Client talks to ZenQuotes over HTTP.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// Random fetches one random motivational quote.
func (c *Client) Random(ctx context.Context) (Quote, error) {
	body, err := c.get(ctx, c.cfg.BaseURL+"/api/random")
	if err != nil {
		return Quote{}, err
	}
	var raw []wireQuote
	if err := json.Unmarshal(body, &raw); err != nil {
		return Quote{}, fmt.Errorf("decode random quote: %w", err)
	}
	if len(raw) == 0 {
		return Quote{}, fmt.Errorf("empty response from /api/random")
	}
	return toQuote(raw[0]), nil
}

// Today fetches today's featured quote. The API returns the same quote for
// the entire day.
func (c *Client) Today(ctx context.Context) (Quote, error) {
	body, err := c.get(ctx, c.cfg.BaseURL+"/api/today")
	if err != nil {
		return Quote{}, err
	}
	var raw []wireQuote
	if err := json.Unmarshal(body, &raw); err != nil {
		return Quote{}, fmt.Errorf("decode today quote: %w", err)
	}
	if len(raw) == 0 {
		return Quote{}, fmt.Errorf("empty response from /api/today")
	}
	return toQuote(raw[0]), nil
}

func toQuote(w wireQuote) Quote {
	return Quote{
		Quote:  w.Q,
		Author: w.A,
		Length: w.C,
	}
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	return min(time.Duration(attempt)*500*time.Millisecond, 5*time.Second)
}
