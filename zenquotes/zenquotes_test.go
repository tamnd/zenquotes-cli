package zenquotes_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/zenquotes-cli/zenquotes"
)

const fakeRandomJSON = `[{"q":"Life is beautiful.","a":"Test Author","h":"<blockquote>Life is beautiful. — Test Author</blockquote>"}]`

const fakeQuotesJSON = `[
  {"q":"The only way to do great work is to love what you do.","a":"Steve Jobs","h":"<blockquote>...</blockquote>"},
  {"q":"In the middle of every difficulty lies opportunity.","a":"Albert Einstein","h":"<blockquote>...</blockquote>"},
  {"q":"It does not matter how slowly you go as long as you do not stop.","a":"Confucius","h":"<blockquote>...</blockquote>"}
]`

func newTestClient(ts *httptest.Server) *zenquotes.Client {
	cfg := zenquotes.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return zenquotes.NewClient(cfg)
}

func TestRandomSendsUA(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = fmt.Fprint(w, fakeRandomJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Random(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent not sent")
	}
}

func TestRandomParsesQuote(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeRandomJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	q, err := c.Random(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if q.Rank != 1 {
		t.Errorf("Rank = %d, want 1", q.Rank)
	}
	if q.Text != "Life is beautiful." {
		t.Errorf("Text = %q, want %q", q.Text, "Life is beautiful.")
	}
	if q.Author != "Test Author" {
		t.Errorf("Author = %q, want %q", q.Author, "Test Author")
	}
}

func TestQuotesParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeQuotesJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Quotes(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	if items[0].Rank != 1 {
		t.Errorf("items[0].Rank = %d, want 1", items[0].Rank)
	}
	if items[0].Text == "" {
		t.Error("items[0].Text is empty")
	}
	if items[0].Author == "" {
		t.Error("items[0].Author is empty")
	}
}

func TestQuotesLimitRespected(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeQuotesJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	items, err := c.Quotes(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
}

func TestQuotesRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeQuotesJSON)
	}))
	defer ts.Close()

	cfg := zenquotes.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := zenquotes.NewClient(cfg)

	_, err := c.Quotes(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}
