package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Cookie represents a simplified cookie for serialization.
type Cookie struct {
	Name     string     `json:"name"`
	Value    string     `json:"value"`
	Domain   string     `json:"domain"`
	Path     string     `json:"path"`
	Expires  *time.Time `json:"expires,omitempty"`
	Secure   bool       `json:"secure"`
	HttpOnly bool       `json:"http_only"`
}

const (
	michelinURL    = "https://guide.michelin.com"
	michelinDomain = "michelin.com"
)

// Login launches a browser, performs login on guide.michelin.com, and returns
// the session cookies scoped to michelin.com. The provided ctx is used to
// cancel the entire operation; timeout additionally caps each element-wait
func Login(ctx context.Context, email, password string, headless bool, timeout time.Duration) ([]Cookie, error) {
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	browser, cleanup, err := launchBrowser(headless)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	browser = browser.Context(ctx)

	page, err := browser.Page(proto.TargetCreateTarget{URL: michelinURL})
	if err != nil {
		return nil, fmt.Errorf("failed to open page: %w", err)
	}

	page = page.Timeout(timeout)

	if err := performLogin(page, email, password); err != nil {
		return nil, err
	}

	return extractMichelinCookies(page)
}

// launchBrowser starts a new browser instance and returns it together with a
// cleanup function that the caller must defer
func launchBrowser(headless bool) (*rod.Browser, func(), error) {
	urlStr, err := launcher.New().Headless(headless).Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(urlStr).MustConnect()
	return browser, func() { _ = browser.Close() }, nil
}

// performLogin drives the multi-step login flow
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
func extractMichelinCookies(page *rod.Page) ([]Cookie, error) {
	rawCookies, err := page.Cookies([]string{michelinURL})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cookies: %w", err)
	}

	var out []Cookie
	for _, c := range rawCookies {
		if !strings.Contains(c.Domain, michelinDomain) {
			continue
		}

		cookie := Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		}
		if c.Expires != 0 {
			t := time.Unix(int64(c.Expires), 0).UTC()
			cookie.Expires = &t
		}
		out = append(out, cookie)
	}

	if len(out) == 0 {
		return nil, errors.New("no michelin.com cookies found after login — credentials may be wrong or the site structure changed")
	}
	return out, nil
}

// WriteConfig serialises cfg as indented JSON to the file at path.
// The file is created with mode 0600 (owner read/write only)
func WriteConfig(path string, cfg any) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}
