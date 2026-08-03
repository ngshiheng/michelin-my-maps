package client

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/storage"
	_ "github.com/mattn/go-sqlite3"
	"github.com/velebak/colly-sqlite3-storage/colly/sqlite3"
)

// sessionCookies is a helper that returns a minimal slice of session cookies.
func sessionCookies(name, value string) []*http.Cookie {
	return []*http.Cookie{{Name: name, Value: value}}
}

// TestNewSeedsCookiesFromSQLite verifies that cookies already stored in SQLite
// are seeded into the in-memory jar when New() starts.
func TestNewSeedsCookiesFromSQLite(t *testing.T) {
	dir := t.TempDir()
	storagePath := filepath.Join(dir, "colly.db")
	domain := "guide.michelin.com"
	target := &url.URL{Scheme: "https", Host: domain}

	// Pre-populate the SQLite storage with a known session cookie, exactly as
	// InitCookies does after a successful login.
	store := &sqlite3.Storage{Filename: storagePath}
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init: %v", err)
	}
	raw := storage.StringifyCookies(sessionCookies("michelin_session", "abc123"))
	store.SetCookies(target, raw)
	store.Close()

	cfg := &Config{
		AllowedDomains: []string{domain},
		StoragePath:    storagePath,
		Delay:          0,
		RandomDelay:    0,
		ThreadCount:    1,
		RequestTimeout: 5 * time.Second,
	}
	cl, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// The seeded cookie must be visible via the collector's jar
	cookies := cl.GetCookies(target.String())
	if cookies["michelin_session"] != "abc123" {
		t.Errorf("expected michelin_session=abc123 in jar, got: %v", cookies)
	}
}

// TestNewStaleRowWinsOnDuplicateCookies documents upstream storage behavior:
// duplicate SetCookies writes keep the oldest value on read.
//
// If this test ever fails (val == "fresh"), the upstream behavior changed and
// cookie reset logic should be re-evaluated.
func TestNewStaleRowWinsOnDuplicateCookies(t *testing.T) {
	dir := t.TempDir()
	storagePath := filepath.Join(dir, "colly.db")
	domain := "guide.michelin.com"
	target := &url.URL{Scheme: "https", Host: domain}

	store := &sqlite3.Storage{Filename: storagePath}
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init: %v", err)
	}

	// Simulate two logins: stale cookie first, then fresh cookie.
	store.SetCookies(target, storage.StringifyCookies(sessionCookies("michelin_session", "stale")))
	store.SetCookies(target, storage.StringifyCookies(sessionCookies("michelin_session", "fresh")))
	store.Close()

	cfg := &Config{
		AllowedDomains: []string{domain},
		StoragePath:    storagePath,
		Delay:          0,
		RandomDelay:    0,
		ThreadCount:    1,
		RequestTimeout: 5 * time.Second,
	}
	cl, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cookies := cl.GetCookies(target.String())
	val := cookies["michelin_session"]
	// Expect "stale": sqlite backend returns the oldest cookie row.
	if val != "stale" {
		t.Errorf("sqlite3 storage returned %q; expected \"stale\" (oldest row wins)", val)
	}
}

// TestMemJarPropagatesRotatedCookies verifies that cookies rotated by server
// responses are sent on the next request from the in-memory jar.
func TestMemJarPropagatesRotatedCookies(t *testing.T) {
	rotations := []string{"SESS-001", "SESS-002", "SESS-003"}
	idx := 0

	// A test server that rotates JSESSIONID on every response.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if idx < len(rotations) {
			http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: rotations[idx]})
			idx++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	cfg := &Config{
		AllowedDomains: []string{u.Hostname()}, // AllowedDomains matches hostname without port
		StoragePath:    filepath.Join(t.TempDir(), "colly.db"),
		Delay:          0,
		RandomDelay:    0,
		ThreadCount:    1,
		RequestTimeout: 5 * time.Second,
	}
	cl, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	cl.collector.WithTransport(srv.Client().Transport)

	// Capture the JSESSIONID sent on each outgoing request.
	// Safe without a mutex: colly.Async(false) means each Visit() completes
	// the full request→response→jar-update cycle before OnRequest fires again.
	var sent []string
	cl.collector.OnRequest(func(r *colly.Request) {
		val := ""
		for _, c := range cl.collector.Cookies(srv.URL) {
			if c.Name == "JSESSIONID" {
				val = c.Value
				break
			}
		}
		sent = append(sent, val)
	})

	for i := range 3 {
		if err := cl.collector.Visit(srv.URL + "/" + string(rune('a'+i))); err != nil {
			t.Fatalf("Visit: %v", err)
		}
	}

	// Request 1: jar is empty before any response        → sends ""
	// Request 2: jar has SESS-001 (from response 1)      → sends "SESS-001"
	// Request 3: jar has SESS-002 (from response 2)      → sends "SESS-002"
	wantSent := []string{"", "SESS-001", "SESS-002"}
	for i, want := range wantSent {
		if sent[i] != want {
			t.Errorf("request %d: sent JSESSIONID=%q, want %q", i+1, sent[i], want)
		}
	}
}

func TestClearCookiesPreservesQueueAndVisited(t *testing.T) {
	dir := t.TempDir()
	storagePath := filepath.Join(dir, "colly.db")

	store := &sqlite3.Storage{Filename: storagePath}
	if err := store.Init(); err != nil {
		t.Fatalf("store.Init: %v", err)
	}

	guideURL := &url.URL{Scheme: "https", Host: "guide.michelin.com"}
	otherURL := &url.URL{Scheme: "https", Host: "example.com"}
	store.SetCookies(guideURL, storage.StringifyCookies(sessionCookies("michelin_session", "stale")))
	store.SetCookies(otherURL, storage.StringifyCookies(sessionCookies("sid", "keep")))
	if err := store.AddRequest([]byte("queued-request")); err != nil {
		t.Fatalf("store.AddRequest: %v", err)
	}
	if err := store.Visited(42); err != nil {
		t.Fatalf("store.Visited: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close: %v", err)
	}

	cl := &Colly{config: &Config{StoragePath: storagePath}}
	if err := cl.ClearCookies("guide.michelin.com"); err != nil {
		t.Fatalf("ClearCookies: %v", err)
	}

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	var cookieGuideCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM cookies WHERE host = ?", "guide.michelin.com").Scan(&cookieGuideCount); err != nil {
		t.Fatalf("count guide cookies: %v", err)
	}
	if cookieGuideCount != 0 {
		t.Fatalf("expected guide cookies to be cleared, got %d", cookieGuideCount)
	}

	var cookieOtherCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM cookies WHERE host = ?", "example.com").Scan(&cookieOtherCount); err != nil {
		t.Fatalf("count other cookies: %v", err)
	}
	if cookieOtherCount != 1 {
		t.Fatalf("expected other host cookies to be preserved, got %d", cookieOtherCount)
	}

	var queueCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM queue").Scan(&queueCount); err != nil {
		t.Fatalf("count queue rows: %v", err)
	}
	if queueCount != 1 {
		t.Fatalf("expected queue rows to be preserved, got %d", queueCount)
	}

	var visitedCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM visited").Scan(&visitedCount); err != nil {
		t.Fatalf("count visited rows: %v", err)
	}
	if visitedCount != 1 {
		t.Fatalf("expected visited rows to be preserved, got %d", visitedCount)
	}
}
