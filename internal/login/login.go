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
	log "github.com/sirupsen/logrus"
)

const (
	michelinURL    = "https://guide.michelin.com"
	michelinDomain = "michelin.com"
	configDir      = ".mym"
	configFile     = "config.json"
)

// Cookie represents a simplified cookie for serialization
type Cookie struct {
	Name     string     `json:"name"`
	Value    string     `json:"value"`
	Domain   string     `json:"domain"`
	Path     string     `json:"path"`
	Expires  *time.Time `json:"expires,omitempty"`
	Secure   bool       `json:"secure"`
	HttpOnly bool       `json:"http_only"`
}

// Config is the canonical shape written to $HOME/.mym/config.json
type Config struct {
	Version   int           `json:"version"`
	CreatedAt time.Time     `json:"created_at"`
	Source    string        `json:"source"`
	Account   AccountConfig `json:"account"`
	Cookies   []Cookie      `json:"cookies"`
}

// AccountConfig holds per-account metadata inside Config
type AccountConfig struct {
	Email     string    `json:"email"`
	LastLogin time.Time `json:"last_login"`
}

// ConfigPath returns the absolute path to $HOME/.mym/config.json.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve home directory: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Login launches a browser, performs login on guide.michelin.com, persists the
// session to $HOME/.mym/config.json, and returns the resulting Config.
// The provided ctx is used to cancel the entire operation; timeout additionally
// caps each element-wait
func Login(ctx context.Context, email, password string, headless bool, timeout time.Duration) error {
	if email == "" || password == "" {
		return errors.New("email and password are required")
	}

	log.WithFields(log.Fields{
		"headless": headless,
		"timeout":  timeout,
	}).Debug("starting login flow")

	browser, cleanup, err := launchBrowser(headless)
	if err != nil {
		return err
	}
	defer cleanup()

	browser = browser.Context(ctx)

	log.WithField("url", michelinURL).Debug("opening page")
	page, err := browser.Page(proto.TargetCreateTarget{URL: michelinURL})
	if err != nil {
		return fmt.Errorf("failed to open page: %w", err)
	}

	page = page.Timeout(timeout)

	if err := performLogin(page, email, password); err != nil {
		log.WithError(err).Error("login flow failed")
		return err
	}

	log.Info("login succeeded, extracting cookies")
	cookies, err := extractMichelinCookies(page)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	cfg := &Config{
		Version:   1,
		CreatedAt: now,
		Source:    "rod",
		Account: AccountConfig{
			Email:     email,
			LastLogin: now,
		},
		Cookies: cookies,
	}

	if err := WriteConfig(cfg); err != nil {
		// Non-fatal: cookies are still valid even if we can't persist them.
		log.WithError(err).Warn("login succeeded but failed to write config")
	}

	return nil
}

// launchBrowser starts a new browser instance and returns it together with a
// cleanup function that the caller must defer
func launchBrowser(headless bool) (*rod.Browser, func(), error) {
	log.WithField("headless", headless).Debug("launching browser")
	urlStr, err := launcher.New().Headless(headless).Launch()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(urlStr).MustConnect()
	log.Debug("browser connected")
	return browser, func() {
		log.Debug("closing browser")
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
		log.WithField("step", step.name).Debug("executing login step")
		if err := step.fn(); err != nil {
			log.WithFields(log.Fields{
				"step": step.name,
			}).WithError(err).Warn("login step failed")
			return fmt.Errorf("login step %q failed: %w", step.name, err)
		}
		log.WithField("step", step.name).Debug("login step completed")
	}
	return nil
}

// clickElement finds an element by XPath, waits for it to be visible, then clicks it
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

// fillInput finds an input by XPath, waits for it to be visible, then types the value
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

// extractMichelinCookies fetches cookies from the page and filters to michelin.com
func extractMichelinCookies(page *rod.Page) ([]Cookie, error) {
	log.WithField("url", michelinURL).Debug("fetching cookies from page")
	rawCookies, err := page.Cookies([]string{michelinURL})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cookies: %w", err)
	}

	log.WithField("total", len(rawCookies)).Debug("raw cookies retrieved, filtering to michelin.com")

	var out []Cookie
	for _, c := range rawCookies {
		if !strings.Contains(c.Domain, michelinDomain) {
			log.WithFields(log.Fields{
				"name":   c.Name,
				"domain": c.Domain,
			}).Debug("skipping cookie (domain mismatch)")
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
		log.WithFields(log.Fields{
			"name":   c.Name,
			"domain": c.Domain,
		}).Debug("accepted cookie")
		out = append(out, cookie)
	}

	if len(out) == 0 {
		log.Warn("no michelin.com cookies found after login")
		return nil, errors.New("no michelin.com cookies found after login — credentials may be wrong or the site structure changed")
	}

	log.WithField("count", len(out)).Info("michelin cookies extracted")
	return out, nil
}

// WriteConfig serialises cfg as indented JSON to $HOME/.mym/config.json.
// It creates the directory if it does not exist
func WriteConfig(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	log.WithField("dir", dir).Debug("ensuring config directory exists")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %q: %w", dir, err)
	}

	log.WithField("path", path).Debug("writing config file")
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

	log.WithField("path", path).Info("config written successfully")
	return nil
}
