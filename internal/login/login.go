package login

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gocolly/colly/v2/storage"
	log "github.com/sirupsen/logrus"
)

const (
	michelinURL    = "https://guide.michelin.com"
	michelinDomain = "michelin.com"
)

// Login logs in via browser and seeds store with the resulting session cookies.
// Pass the same store to colly's collector.SetStorage() for downstream scraping.
func Login(ctx context.Context, email, password string, headless bool, timeout time.Duration, store storage.Storage) error {
	if email == "" || password == "" {
		return errors.New("email and password are required")
	}
	if store == nil {
		return errors.New("storage is required")
	}

	if err := store.Init(); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	browser, cleanup, err := launchBrowser(headless)
	if err != nil {
		return err
	}
	defer cleanup()

	page, err := browser.Context(ctx).Page(proto.TargetCreateTarget{URL: michelinURL})
	if err != nil {
		return fmt.Errorf("failed to open page: %w", err)
	}

	if err := performLogin(page.Timeout(timeout), email, password); err != nil {
		log.WithError(err).Error("login flow failed")
		return err
	}

	cookies, err := extractMichelinCookies(page)
	if err != nil {
		return err
	}

	// colly-sqlite3-storage appends cookie rows, so clear stale state when supported.
	if clearable, ok := store.(interface{ Clear() error }); ok {
		if err := clearable.Clear(); err != nil {
			log.WithError(err).Warn("failed to clear storage before session seed")
		}
	}
	if err := store.Init(); err != nil {
		return fmt.Errorf("failed to reinitialize storage: %w", err)
	}

	u, _ := url.Parse(michelinURL)
	lines := make([]string, len(cookies))
	for i, c := range cookies {
		lines[i] = c.String()
	}
	store.SetCookies(u, strings.Join(lines, "\n"))

	log.WithField("cookie_count", len(cookies)).Info("session stored")
	return nil
}

// launchBrowser starts a new browser instance and returns it together with a
// cleanup function that the caller must defer.
func launchBrowser(headless bool) (*rod.Browser, func(), error) {
	urlStr, err := launcher.New().Headless(headless).Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(urlStr).MustConnect()
	return browser, func() {
		_ = browser.Close()
	}, nil
}

// performLogin drives the multi-step login flow.
func performLogin(page *rod.Page, email, password string) error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"click profile icon", func() error {
			return clickElement(page, "//img[contains(@class,'js-img-profile-menu')]")
		}},
		{"click login link", func() error {
			return clickElement(page, "//a[contains(normalize-space(.),'Login') and contains(@class,'js-auth__social-sign-in-button')]")
		}},
		{"fill email", func() error {
			return fillInput(page, "//div[@class='form-group']//input[@id='emailId' or @name='email']", email)
		}},
		{"click continue", func() error {
			return clickElement(page, "//button[contains(normalize-space(.),'Continue') and contains(@class,'js-auth__sign-in-continue-button')]")
		}},
		{"fill password", func() error {
			return fillInput(page, "//input[@type='password' and contains(@class,'js-account-pass')]", password)
		}},
		{"click sign in", func() error {
			return clickElement(page, "//button[contains(normalize-space(.),'Sign In') and contains(@class,'js-auth__sign-in-button')]")
		}},
		{"wait for idle", func() error {
			return page.WaitIdle(2 * time.Second)
		}},
	}

	for _, step := range steps {
		if err := step.fn(); err != nil {
			return fmt.Errorf("login step %q failed: %w", step.name, err)
		}
	}
	return nil
}

// clickElement finds an element by XPath, waits for it to be visible, then clicks it.
func clickElement(page *rod.Page, xpath string) error {
	el, err := page.ElementX(xpath)
	if err != nil {
		return fmt.Errorf("element not found (%s): %w", xpath, err)
	}
	if err := el.WaitVisible(); err != nil {
		return fmt.Errorf("element not visible (%s): %w", xpath, err)
	}
	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click failed (%s): %w", xpath, err)
	}
	return nil
}

// fillInput finds an input by XPath, waits for it to be visible, then types the value.
func fillInput(page *rod.Page, xpath, value string) error {
	el, err := page.ElementX(xpath)
	if err != nil {
		return fmt.Errorf("input not found (%s): %w", xpath, err)
	}
	if err := el.WaitVisible(); err != nil {
		return fmt.Errorf("input not visible (%s): %w", xpath, err)
	}
	if err := el.Input(value); err != nil {
		return fmt.Errorf("input failed (%s): %w", xpath, err)
	}
	return nil
}

// extractMichelinCookies fetches cookies from the page and filters to michelin.com.
func extractMichelinCookies(page *rod.Page) ([]*http.Cookie, error) {
	rawCookies, err := page.Cookies([]string{michelinURL})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cookies: %w", err)
	}

	out := make([]*http.Cookie, 0, len(rawCookies))
	for _, c := range rawCookies {
		if !strings.Contains(c.Domain, michelinDomain) {
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

	if len(out) == 0 {
		return nil, errors.New("no michelin.com cookies after login: credentials may be wrong or site structure changed")
	}
	return out, nil
}
