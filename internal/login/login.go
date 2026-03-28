package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// Cookie represents a simplified cookie for serialization.
type Cookie struct {
	Name     string     `json:"name"`
	Value    string     `json:"value"`
	Domain   string     `json:"domain"`
	Path     string     `json:"path"`
	Expires  *time.Time `json:"expires"`
	Secure   bool       `json:"secure"`
	HttpOnly bool       `json:"http_only"`
}

// Login launches a browser, performs login using XPath selectors only, and returns cookies for the domain.
func Login(ctx context.Context, email, password string, headless bool, timeout time.Duration) ([]Cookie, error) {
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	u := launcher.New()
	if !headless {
		u = u.Headless(false)
	} else {
		u = u.Headless(true)
	}

	urlStr, err := u.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(urlStr).MustConnect()
	defer func() {
		_ = browser.Close()
	}()

	page := browser.MustPage("https://guide.michelin.com")
	page = page.Timeout(timeout)

	// Use XPath exclusively as requested
	// 1. Click profile icon
	if err := page.MustElementX("//img[contains(@class,'js-img-profile-menu')]").WaitVisible(); err == nil {
		_ = page.MustElementX("//img[contains(@class,'js-img-profile-menu')]").MustClick()
	} else {
		// try clicking the icon even if wait failed
		_ = page.MustElementX("//img[contains(@class,'js-img-profile-menu')]").MustClick()
	}

	// 2. Click 'Login' link
	page.MustElementX("//a[contains(normalize-space(.),'Login') and contains(@class,'js-auth__social-sign-in-button')]").MustClick()

	// 3. Fill email
	emailXPath := "//div[@class='form-group']//input[@id='emailId' or @name='email']"
	page.MustElementX(emailXPath).Input(email)

	// 4. Click Continue
	page.MustElementX("//button[contains(normalize-space(.),'Continue') and contains(@class,'js-auth__sign-in-continue-button')]").MustClick()

	// 5. Fill password
	page.MustElementX("//input[@type='password' and contains(@class,'js-account-pass')]").Input(password)

	// 6. Click Sign In
	page.MustElementX("//button[contains(normalize-space(.),'Sign In') and contains(@class,'js-auth__sign-in-button')]").MustClick()

	// Wait for navigation or cookie set; sleep briefly then fetch cookies
	time.Sleep(2 * time.Second)

	// Fetch cookies from browser
	rawCookies, err := page.Cookies([]string{"https://guide.michelin.com"})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cookies: %w", err)
	}

	var out []Cookie
	for _, c := range rawCookies {
		// filter domain to guide.michelin.com or contains
		u, _ := url.Parse(c.Domain)
		_ = u // ignore parse error
		// Accept cookies where domain contains "michelin.com"
		if c.Domain == "" || !(containsIgnoreCase(c.Domain, "michelin.com") || containsIgnoreCase(c.Domain, "guide.michelin.com")) {
			continue
		}
		var exp *time.Time
		if c.Expires != 0 {
			t := time.Unix(int64(c.Expires), 0).UTC()
			exp = &t
		}
		out = append(out, Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  exp,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		})
	}

	if len(out) == 0 {
		return nil, errors.New("no michelin cookies found after login")
	}

	// Optionally write a debug file (not by default). Return cookies.
	return out, nil
}

func containsIgnoreCase(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || s == sub)
}

// Helper to write config (used by CLI)
func WriteConfig(path string, cfg interface{}) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}
