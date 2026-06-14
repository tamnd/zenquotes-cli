package zenquotes_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamnd/zenquotes-cli/zenquotes"
)

const fakeRandomJSON = `[{"q":"Life is beautiful.","a":"Test Author","c":"18","h":"<blockquote>Life is beautiful. — Test Author</blockquote>"}]`

const fakeTodayJSON = `[{"q":"Today is a good day to learn.","a":"Someone Wise","c":"30","h":"<blockquote>...</blockquote>"}]`

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
	if q.Quote != "Life is beautiful." {
		t.Errorf("Quote = %q, want %q", q.Quote, "Life is beautiful.")
	}
	if q.Author != "Test Author" {
		t.Errorf("Author = %q, want %q", q.Author, "Test Author")
	}
	if q.Length != "18" {
		t.Errorf("Length = %q, want %q", q.Length, "18")
	}
}

func TestTodayParsesQuote(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, fakeTodayJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	q, err := c.Today(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if q.Quote != "Today is a good day to learn." {
		t.Errorf("Quote = %q, want %q", q.Quote, "Today is a good day to learn.")
	}
	if q.Author != "Someone Wise" {
		t.Errorf("Author = %q, want %q", q.Author, "Someone Wise")
	}
	if q.Length != "30" {
		t.Errorf("Length = %q, want %q", q.Length, "30")
	}
}

func TestRandomRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, fakeRandomJSON)
	}))
	defer ts.Close()

	cfg := zenquotes.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 3
	c := zenquotes.NewClient(cfg)

	_, err := c.Random(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}

func TestTodayHitsCorrectPath(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = fmt.Fprint(w, fakeTodayJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Today(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/today" {
		t.Errorf("path = %q, want /api/today", gotPath)
	}
}

func TestRandomHitsCorrectPath(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = fmt.Fprint(w, fakeRandomJSON)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Random(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/random" {
		t.Errorf("path = %q, want /api/random", gotPath)
	}
}
