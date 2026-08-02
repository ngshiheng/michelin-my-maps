package session

import (
	"net/http"
	"testing"

	"github.com/go-rod/rod/lib/proto"
)

func TestFilterCookiesKeepsMichelinDomains(t *testing.T) {
	rawCookies := []*proto.NetworkCookie{
		{Name: "michelin_session", Value: "abc", Domain: "guide.michelin.com", Path: "/"},
		{Name: "session", Value: "xyz", Domain: "example.com", Path: "/"},
		{Name: "locale", Value: "en-SG", Domain: ".michelin.com", Path: "/"},
	}

	got := filterCookies(rawCookies)
	if len(got) != 2 {
		t.Fatalf("expected 2 Michelin cookies, got %d", len(got))
	}

	if got[0].Name != "michelin_session" || got[1].Name != "locale" {
		t.Fatalf("unexpected cookie order: %+v", got)
	}

	if got[0].Value != "abc" {
		t.Fatalf("expected filtered cookie value to be preserved, got %q", got[0].Value)
	}

	var _ []*http.Cookie = got
}
