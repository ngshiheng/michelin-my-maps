package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	log "github.com/sirupsen/logrus"
)

const (
	michelinURL    = "https://guide.michelin.com/sg/en"
	michelinDomain = "michelin.com"

	xPathProfileIcon = "//img[contains(@class,'js-img-profile-menu')]"
	xPathLoginLink   = "//a[contains(text(), 'Sign In')]"
	xPathEmailInput  = "//input[@id='emailId']"
	xPathContinueBtn = "//button[contains(text(), 'Continue')]"
	xPathPassword    = "//input[@name='password' and @type='password']"
	xPathSignInBtn   = "//button[contains(text(), 'Sign In')]"
)

// Login logs in via browser and returns the resulting michelin.com session cookies
func Login(ctx context.Context, email, password string, headless bool, timeout time.Duration) ([]*http.Cookie, error) {
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	browser, cleanup, err := launchBrowser(headless)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	page, err := browser.Context(ctx).Page(proto.TargetCreateTarget{URL: michelinURL})
	if err != nil {
		return nil, fmt.Errorf("failed to open page: %w", err)
	}

	if err := performLogin(page.Timeout(timeout), email, password); err != nil {
		log.WithError(err).Error("login flow failed")
		return nil, err
	}

	return extractCookies(page)
}

// launchBrowser starts a new browser instance and returns it together with a
// cleanup function that the caller must defer
func launchBrowser(headless bool) (*rod.Browser, func(), error) {
	l := launcher.New().Headless(headless)

	browserBin := os.Getenv("MYM_BROWSER_BIN")
	if browserBin != "" {
		l = l.Bin(browserBin)
	}

	noSandbox := os.Getenv("MYM_NO_SANDBOX") == "1"
	if noSandbox {
		l = l.NoSandbox(true)
	}

	log.WithFields(log.Fields{
		"browser_bin": browserBin,
		"headless":    headless,
		"no_sandbox":  noSandbox,
	}).Info("launching browser")

	urlStr, err := l.Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to launch browser (bin=%q): %w", browserBin, err)
	}

	browser := rod.New().ControlURL(urlStr)
	if err := browser.Connect(); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to browser (bin=%q): %w", browserBin, err)
	}
	log.Debug("browser connected")

	return browser, func() {
		_ = browser.Close()
	}, nil
}

// performLogin drives the multi-step login flow
func performLogin(page *rod.Page, email, password string) error {
	steps := []struct {
		name string
		fn   func() error
	}{
		{"click profile icon", func() error {
			return clickElement(page, xPathProfileIcon)
		}},
		{"click login link", func() error {
			return clickElement(page, xPathLoginLink)
		}},
		{"fill email", func() error {
			return fillInput(page, xPathEmailInput, email)
		}},
		{"click continue", func() error {
			return clickElement(page, xPathContinueBtn)
		}},
		{"fill password", func() error {
			return fillInput(page, xPathPassword, password)
		}},
		{"click sign in", func() error {
			return clickElement(page, xPathSignInBtn)
		}},
		{"wait for idle", func() error {
			return page.WaitIdle(2 * time.Second)
		}},
	}

	for _, step := range steps {
		log.WithField("step", step.name).Debug("executing login step")
		if err := step.fn(); err != nil {
			return fmt.Errorf("login step %q failed: %w", step.name, err)
		}
		log.WithField("step", step.name).Debug("login step completed")
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

// extractCookies fetches cookies from the page and filters to michelin.com
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

	if len(out) == 0 {
		return nil, errors.New("no michelin.com cookies after login: credentials may be wrong or site structure changed")
	}
	return out, nil
}
