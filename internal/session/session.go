package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const (
	michelinURL    = "https://guide.michelin.com/sg/en"
	michelinDomain = "michelin.com"
)

// GetCookies opens a browser window and waits for Michelin session cookies
func GetCookies(ctx context.Context, timeout time.Duration) ([]*http.Cookie, error) {
	browser, cleanup, err := launchBrowser()
	if err != nil {
		return nil, err
	}
	defer cleanup()

	page, err := browser.Context(ctx).Page(proto.TargetCreateTarget{URL: michelinURL})
	if err != nil {
		return nil, fmt.Errorf("failed to open page: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for {
		cookies, err := extractCookies(page)
		if err == nil {
			return cookies, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for Michelin cookies: %w", err)
		}
		slog.Debug("waiting for Michelin cookies", "error", err)
		time.Sleep(2 * time.Second)
	}
}

func launchBrowser() (*rod.Browser, func(), error) {
	l := launcher.New().Headless(true)

	browserBin := os.Getenv("MYM_BROWSER_BIN")
	if browserBin != "" {
		l = l.Bin(browserBin)
	}

	noSandbox := os.Getenv("MYM_NO_SANDBOX") == "1"
	if noSandbox {
		l = l.NoSandbox(true)
	}

	slog.Info("launching browser", "browser_bin", browserBin, "no_sandbox", noSandbox)

	urlStr, err := l.Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to launch browser (bin=%q): %w", browserBin, err)
	}

	browser := rod.New().ControlURL(urlStr)
	if err := browser.Connect(); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to browser (bin=%q): %w", browserBin, err)
	}
	slog.Debug("browser connected")

	return browser, func() {
		_ = browser.Close()
	}, nil
}

func extractCookies(page *rod.Page) ([]*http.Cookie, error) {
	info, err := page.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to read page info for cookie retrieval: %w", err)
	}

	requestURLs := []string{michelinURL}
	if info != nil && strings.TrimSpace(info.URL) != "" {
		requestURLs = append(requestURLs, info.URL)
	}

	rawCookies, err := page.Cookies(requestURLs)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cookies: %w", err)
	}

	out := filterCookies(rawCookies)
	if len(out) == 0 {
		return nil, errors.New("no michelin.com cookies found yet")
	}
	return out, nil
}

func filterCookies(rawCookies []*proto.NetworkCookie) []*http.Cookie {
	out := make([]*http.Cookie, 0, len(rawCookies))
	for _, c := range rawCookies {
		normalizedDomain := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(c.Domain)), ".")
		if normalizedDomain != michelinDomain && !strings.HasSuffix(normalizedDomain, "."+michelinDomain) {
			continue
		}

		hc := &http.Cookie{
			Name:     c.Name,
			Value:    strings.ReplaceAll(c.Value, `"`, ""), // strip invalid quote chars
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		}
		if c.Expires != 0 {
			hc.Expires = time.Unix(int64(c.Expires), 0).UTC()
		}
		out = append(out, hc)
	}
	return out
}
