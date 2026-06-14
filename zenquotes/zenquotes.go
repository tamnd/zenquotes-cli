// Package zenquotes is the library behind the zenquotes command line:
// the HTTP client, request shaping, and the typed data models for the
// ZenQuotes.io public motivational quote API.
//
// The API requires no authentication. A polite User-Agent and 200 ms pacing
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
		BaseURL:   "https://zenquotes.io/api",
		UserAgent: "zenquotes-cli/0.1 (github.com/tamnd/zenquotes-cli)",
		Rate:      200 * time.Millisecond,
		Timeout:   15 * time.Second,
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
	u := c.cfg.BaseURL + "/random"
	body, err := c.get(ctx, u)
	if err != nil {
		return Quote{}, err
	}
	var raw []rawQuote
	if err := json.Unmarshal(body, &raw); err != nil {
		return Quote{}, fmt.Errorf("decode random quote: %w", err)
	}
	if len(raw) == 0 {
		return Quote{}, fmt.Errorf("empty response from /random")
	}
	return Quote{Rank: 1, Text: raw[0].Q, Author: raw[0].A}, nil
}

// Quotes fetches a batch of up to 50 motivational quotes.
// Pass limit <= 0 to return all 50.
func (c *Client) Quotes(ctx context.Context, limit int) ([]Quote, error) {
	u := c.cfg.BaseURL + "/quotes"
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var raw []rawQuote
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode quotes: %w", err)
	}
	items := make([]Quote, 0, len(raw))
	for i, r := range raw {
		items = append(items, Quote{Rank: i + 1, Text: r.Q, Author: r.A})
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items, nil
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
