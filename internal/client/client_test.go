package client

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/storage"
	"github.com/velebak/colly-sqlite3-storage/colly/sqlite3"
)

// sessionCookies is a helper that returns a minimal slice of session cookies.
func sessionCookies(name, value string) []*http.Cookie {
	return []*http.Cookie{{Name: name, Value: value}}
}

// TestNewSeedsCookiesFromSQLite verifies that cookies already stored in the
// SQLite backend are seeded into the in-memory jar when New() is called.
// This guards against the colly-sqlite3-storage plain-INSERT bug where the
// stale first row would be returned on every read, causing the scraper to send
// expired session cookies after a re-login.
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

// TestNewStaleRowWinsOnDuplicateCookies documents the colly-sqlite3-storage
// plain-INSERT bug: when two SetCookies calls write the same cookie name,
// Cookies() returns only the first (stale) row — the latest value is silently
// discarded. New() seeds memJar from that stale row, so the scraper sends an
// expired session cookie after a re-login. This is exactly the bug that
// InitCookies (Clear+Init) is designed to prevent.
//
// If this test ever fails (val == "fresh"), the library has been fixed and
// the InitCookies workaround can be removed.
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
	// Expect "stale": the plain-INSERT bug means Cookies() returns the first row.
	// A failure here means the library now returns "fresh" — the bug is fixed.
	if val != "stale" {
		t.Errorf("sqlite3 storage returned %q; expected \"stale\" (plain-INSERT bug) — if \"fresh\", the bug is fixed and InitCookies workaround can be removed", val)
	}
}

// TestMemJarPropagatesRotatedCookies is the programmatic proof that the
// in-memory jar correctly propagates server-rotated session cookies across
// successive requests — the behaviour confirmed by the debug logs on
// 2026-04-03. guide.michelin.com rotates JSESSIONID on every response; this
// test verifies that each outgoing request carries the value received from the
// immediately preceding response.
//
// How the proof was collected in production:
//
//  1. Added a temporary OnRequest hook to client.New() and GetDetailCollector()
//     that logged the JSESSIONID value from memJar at the moment each request
//     was dispatched.
//
//  2. Added a temporary OnResponse hook that logged the Set-Cookie header
//     (with cache_hit context to filter out replayed cached headers).
//
//  3. Ran `mym scrape --log debug` and observed the following repeating pattern
//     in the output — every received value appeared as the outgoing value on
//     the very next request:
//
//     DEBU received JSESSIONID cookie  Set-Cookie="JSESSIONID=18A457F4..." cache_hit=false url=".../les-plats-canailles..."
//     DEBU outgoing request cookies    JSESSIONID=18A457F4...              url=".../l-amandier..."         ← matches ✓
//     DEBU received JSESSIONID cookie  Set-Cookie="JSESSIONID=218539931D..." cache_hit=false url=".../l-amandier..."
//     DEBU outgoing request cookies    JSESSIONID=218539931D...             url=".../atelier-de-bossime..."  ← matches ✓
//     DEBU received JSESSIONID cookie  Set-Cookie="JSESSIONID=99DEE023..."  cache_hit=false url=".../atelier-de-bossime..."
//     DEBU outgoing request cookies    JSESSIONID=99DEE023...               url=".../mout..."               ← matches ✓
//
// The test below reproduces this with a local httptest server so it can be
// verified deterministically without network access.
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
