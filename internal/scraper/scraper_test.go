package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/config"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/models"
	"github.com/ngshiheng/michelin-my-maps/v3/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestRestaurantDetailExtraction tests complete restaurant detail page processing
func TestRestaurantDetailExtraction(t *testing.T) {
	htmlContent := loadTestHTML(t, "restaurant_detail.html")
	server := createTestServer(htmlContent)
	defer server.Close()

	t.Run("extracts all restaurant data correctly", func(t *testing.T) {
		cfg := createTestConfig()
		scraper := &Scraper{config: cfg}

		c := colly.NewCollector()
		var extractedData storage.RestaurantData

		c.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
			e.Request.Ctx.Put("location", "Singapore")
			e.Request.Ctx.Put("latitude", "1.304144")
			e.Request.Ctx.Put("longitude", "103.83147")

			extractedData = scraper.extractRestaurantData(e)
		})

		c.Visit(server.URL)

		// Verify all extracted data
		expected := storage.RestaurantData{
			Name:                  "Les Amis",
			Address:               "Shaw Centre, #01-16, 1 Scotts Road, 228208, Singapore",
			Location:              "Singapore",
			Latitude:              "1.304144",
			Longitude:             "103.83147",
			Cuisine:               "French",
			Price:                 "$$$$",
			Distinction:           "3 Stars",
			Description:           "Chef Sebastien Lepinoy's contemporary French cuisine showcases premium ingredients with refined techniques. The restaurant offers an intimate dining experience with impeccable service.",
			PhoneNumber:           "+6567332225",
			WebsiteURL:            "https://www.lesamis.com.sg/",
			FacilitiesAndServices: "Air conditioning,Car park,Interesting wine list,Restaurant offering vegetarian menus,Valet parking,Wheelchair access",
			GreenStar:             false,
		}

		assert.Equal(t, expected.Name, extractedData.Name)
		assert.Equal(t, expected.Distinction, extractedData.Distinction)
		assert.Equal(t, expected.Cuisine, extractedData.Cuisine)
		assert.Equal(t, expected.Price, extractedData.Price)
		assert.Equal(t, expected.PhoneNumber, extractedData.PhoneNumber)
		assert.Equal(t, expected.WebsiteURL, extractedData.WebsiteURL)
		assert.Equal(t, expected.Address, extractedData.Address)
		assert.Equal(t, expected.FacilitiesAndServices, extractedData.FacilitiesAndServices)
		assert.Equal(t, expected.GreenStar, extractedData.GreenStar)
	})

	t.Run("validates XPath selectors for detail page", func(t *testing.T) {
		tests := []struct {
			description string
			xpath       string
			expectCount int
			expectText  string
		}{
			{
				description: "extracts restaurant name",
				xpath:       restaurantNameXPath,
				expectCount: 2,
				expectText:  "Les Amis",
			},
			{
				description: "extracts restaurant address",
				xpath:       restaurantAddressXPath,
				expectCount: 1,
				expectText:  "Shaw Centre, #01-16, 1 Scotts Road, 228208, Singapore",
			},
			{
				description: "extracts facilities list",
				xpath:       restaurantFacilitiesAndServicesXPath,
				expectCount: 6,
			},
		}

		for _, tt := range tests {
			t.Run(tt.description, func(t *testing.T) {
				c := colly.NewCollector()
				var results []string

				c.OnXML(tt.xpath, func(e *colly.XMLElement) {
					results = append(results, e.Text)
				})

				c.Visit(server.URL)

				assert.Len(t, results, tt.expectCount)
				if tt.expectText != "" && len(results) > 0 {
					assert.Contains(t, results[0], tt.expectText)
				}
			})
		}
	})
}

// TestRestaurantListProcessing tests restaurant list page processing and navigation
func TestRestaurantListProcessing(t *testing.T) {
	htmlContent := loadTestHTML(t, "restaurant_list.html")
	server := createTestServer(htmlContent)
	defer server.Close()

	t.Run("extracts restaurant URLs and metadata from list", func(t *testing.T) {
		c := colly.NewCollector()
		var extractedURLs []string
		var extractedLocations []string
		var coordinatePairs []string

		c.OnXML(restaurantXPath, func(e *colly.XMLElement) {
			// Extract URL
			url := e.ChildAttr(restaurantDetailUrlXPath, "href")
			if url != "" {
				extractedURLs = append(extractedURLs, url)
			}

			// Extract location
			location := e.ChildText(restaurantLocationXPath)
			if location != "" {
				extractedLocations = append(extractedLocations, location)
			}

			// Extract coordinates
			longitude := e.Attr("data-lng")
			latitude := e.Attr("data-lat")
			if longitude != "" && latitude != "" {
				coordinatePairs = append(coordinatePairs, fmt.Sprintf("%s,%s", latitude, longitude))
			}
		})

		c.Visit(server.URL)

		// Verify we extracted data
		assert.Greater(t, len(extractedURLs), 0, "Should extract restaurant URLs from list page")
		assert.Greater(t, len(extractedLocations), 0, "Should extract restaurant locations from list page")

		t.Logf("Extracted %d URLs, %d locations, %d coordinate pairs",
			len(extractedURLs), len(extractedLocations), len(coordinatePairs))

		// Verify URLs look like restaurant detail URLs
		for _, url := range extractedURLs[:min(3, len(extractedURLs))] {
			assert.Contains(t, url, "restaurant", "URL should contain 'restaurant': %s", url)
		}
	})

	t.Run("validates XPath selectors for list page", func(t *testing.T) {
		tests := []struct {
			description    string
			xpath          string
			expectMinCount int
		}{
			{
				description:    "finds restaurant cards",
				xpath:          restaurantXPath,
				expectMinCount: 1,
			},
			{
				description:    "finds detail page links",
				xpath:          restaurantDetailUrlXPath,
				expectMinCount: 1,
			},
			{
				description:    "finds restaurant locations",
				xpath:          restaurantLocationXPath,
				expectMinCount: 1,
			},
			{
				description:    "handles pagination links",
				xpath:          nextPageArrowButtonXPath,
				expectMinCount: 0, // May or may not have next page
			},
		}

		for _, tt := range tests {
			t.Run(tt.description, func(t *testing.T) {
				c := colly.NewCollector()
				var results []string

				c.OnXML(tt.xpath, func(e *colly.XMLElement) {
					results = append(results, e.Text)
				})

				c.Visit(server.URL)

				assert.GreaterOrEqual(t, len(results), tt.expectMinCount, tt.description)
				t.Logf("Found %d elements for: %s", len(results), tt.description)
			})
		}
	})
}

// TestScraperIntegration tests end-to-end scraper workflow
func TestScraperIntegration(t *testing.T) {
	t.Run("processes detail page with complete workflow", func(t *testing.T) {
		htmlContent := loadTestHTML(t, "restaurant_detail.html")
		server := createTestServer(htmlContent)
		defer server.Close()

		cfg := createTestConfig()
		mockRepo := &MockRepository{}
		mockRepo.On("UpsertRestaurantWithAward", context.Background(), mock.Anything).Return(nil)

		client, err := newWebClient(cfg)
		require.NoError(t, err)

		scraper := New(client, mockRepo, cfg)

		// Test the detail page extraction workflow
		c := colly.NewCollector()
		c.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
			e.Request.Ctx.Put("location", "Singapore")
			e.Request.Ctx.Put("latitude", "1.304144")
			e.Request.Ctx.Put("longitude", "103.83147")

			data := scraper.extractRestaurantData(e)
			err := mockRepo.UpsertRestaurantWithAward(context.Background(), data)
			assert.NoError(t, err)
		})

		c.Visit(server.URL)
		mockRepo.AssertExpectations(t)
	})

	t.Run("processes list page with minimal extraction", func(t *testing.T) {
		htmlContent := loadTestHTML(t, "restaurant_list.html")
		server := createTestServer(htmlContent)
		defer server.Close()

		cfg := createTestConfig()
		mockRepo := &MockRepository{}
		mockRepo.On("UpsertRestaurantWithAward", context.Background(), mock.Anything).Return(nil).Maybe()

		client, err := newWebClient(cfg)
		require.NoError(t, err)

		scraper := New(client, mockRepo, cfg)

		// Test that list page doesn't cause errors
		c := colly.NewCollector()
		c.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
			e.Request.Ctx.Put("location", "Singapore")
			e.Request.Ctx.Put("latitude", "1.304144")
			e.Request.Ctx.Put("longitude", "103.83147")

			data := scraper.extractRestaurantData(e)
			err := mockRepo.UpsertRestaurantWithAward(context.Background(), data)
			assert.NoError(t, err)
		})

		c.Visit(server.URL)
		// No strict expectations for list page
	})
}

// TestErrorHandling tests scraper resilience to various error conditions
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		description string
		html        string
		expectError bool
	}{
		{
			description: "handles malformed HTML gracefully",
			html:        `<div class="incomplete-tag"><span>Missing closing tags`,
			expectError: false,
		},
		{
			description: "handles missing required elements",
			html:        `<html><body><div>No restaurant data here</div></body></html>`,
			expectError: false,
		},
		{
			description: "handles invalid coordinate data",
			html:        `<div class="restaurant-card" data-lat="invalid" data-lng="also-invalid">Restaurant</div>`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			server := createTestServer(tt.html)
			defer server.Close()

			cfg := createTestConfig()
			mockRepo := &MockRepository{}
			mockRepo.On("UpsertRestaurantWithAward", context.Background(), mock.Anything).Return(nil)

			scraper, err := createTestScraper(cfg, mockRepo, server.URL)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err = scraper.Crawl(ctx)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper functions

func createTestServer(htmlContent string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, htmlContent)
	}))
}

func loadTestHTML(t *testing.T, filename string) string {
	testdataPath := filepath.Join("testdata", filename)
	content, err := os.ReadFile(testdataPath)
	require.NoError(t, err, "Failed to load test HTML file: %s", filename)
	return string(content)
}

func createTestConfig() *config.Config {
	cfg := config.Default()
	cfg.Scraper.Delay = 100 * time.Millisecond // Fast for testing
	cfg.Scraper.AdditionalRandomDelay = 0
	cfg.Scraper.MaxRetry = 1
	cfg.Cache.Path = "test_cache"
	return cfg
}

func createTestScraper(cfg *config.Config, repo storage.RestaurantRepository, serverURL string) (*Scraper, error) {
	client, err := newWebClient(cfg)
	if err != nil {
		return nil, err
	}

	scraper := New(client, repo, cfg)

	// Replace the default URLs with our test server URL
	scraper.michelinURLs = []models.GuideURL{
		{
			Distinction: "Test",
			URL:         serverURL,
		},
	}

	return scraper, nil
}

// MockRepository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) SaveRestaurant(ctx context.Context, restaurant *models.Restaurant) error {
	args := m.Called(ctx, restaurant)
	return args.Error(0)
}

func (m *MockRepository) FindRestaurantByURL(ctx context.Context, url string) (*models.Restaurant, error) {
	args := m.Called(ctx, url)
	return args.Get(0).(*models.Restaurant), args.Error(1)
}

func (m *MockRepository) SaveAward(ctx context.Context, award *models.RestaurantAward) error {
	args := m.Called(ctx, award)
	return args.Error(0)
}

func (m *MockRepository) FindAwardByRestaurantAndYear(ctx context.Context, restaurantID uint, year int) (*models.RestaurantAward, error) {
	args := m.Called(ctx, restaurantID, year)
	return args.Get(0).(*models.RestaurantAward), args.Error(1)
}

func (m *MockRepository) UpdateAward(ctx context.Context, award *models.RestaurantAward) error {
	args := m.Called(ctx, award)
	return args.Error(0)
}

func (m *MockRepository) UpsertRestaurantWithAward(ctx context.Context, restaurantData storage.RestaurantData) error {
	args := m.Called(ctx, restaurantData)
	return args.Error(0)
}
