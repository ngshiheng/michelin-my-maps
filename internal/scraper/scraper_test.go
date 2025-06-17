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

	expected := storage.RestaurantData{
		Name:                  "Les Amis",
		Address:               "Shaw Centre, #01-16, 1 Scotts Road, 228208, Singapore",
		Location:              "Singapore",
		Latitude:              "1.304144",
		Longitude:             "103.83147",
		Cuisine:               "French",
		Price:                 "$$$$",
		Distinction:           "3 Stars",
		Description:           "Haute cuisine is increasingly about the chef embracing creative freedom and dictating the whole experience. At the longstanding, singularly sophisticated and world-renowned Les Amis, the choice is yours. The prix-fixe menus present a wide range of modern French classics with occasional Asian twists, in which simple ingredient combinations and seasoning highlight the natural flavours. Consider booking the chef's table for an intimate experience.",
		PhoneNumber:           "+6567332225",
		WebsiteURL:            "https://www.lesamis.com.sg/",
		FacilitiesAndServices: "Air conditioning,Car park,Interesting wine list,Restaurant offering vegetarian menus,Valet parking,Wheelchair access",
		GreenStar:             false,
	}

	assert.Equal(t, expected.Name, extractedData.Name)
	assert.Equal(t, expected.Address, extractedData.Address)
	assert.Equal(t, expected.Location, extractedData.Location)
	assert.Equal(t, expected.Latitude, extractedData.Latitude)
	assert.Equal(t, expected.Longitude, extractedData.Longitude)
	assert.Equal(t, expected.Cuisine, extractedData.Cuisine)
	assert.Equal(t, expected.Price, extractedData.Price)
	assert.Equal(t, expected.Distinction, extractedData.Distinction)
	assert.Equal(t, expected.Description, extractedData.Description)
	assert.Equal(t, expected.PhoneNumber, extractedData.PhoneNumber)
	assert.Equal(t, expected.WebsiteURL, extractedData.WebsiteURL)
	assert.Equal(t, expected.FacilitiesAndServices, extractedData.FacilitiesAndServices)
	assert.Equal(t, expected.GreenStar, extractedData.GreenStar)
}

// TestRestaurantListExtraction tests complete restaurant list page processing
func TestRestaurantListExtraction(t *testing.T) {
	htmlContent := loadTestHTML(t, "restaurant_list.html")
	server := createTestServer(htmlContent)
	defer server.Close()

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

	// Verify we extracted the expected data
	assert.Equal(t, 24, len(extractedURLs), "Should extract 24 restaurant URLs")
	assert.Equal(t, 24, len(extractedLocations), "Should extract 24 restaurant locations")
	assert.Equal(t, 24, len(coordinatePairs), "Should extract 24 coordinate pairs")

	// Verify URL format (sample check)
	for _, url := range extractedURLs[:min(3, len(extractedURLs))] {
		assert.Contains(t, url, "restaurant", "URL should contain 'restaurant': %s", url)
		assert.NotEmpty(t, url, "URL should not be empty")
	}

	// Verify location format (sample check)
	for _, location := range extractedLocations[:min(3, len(extractedLocations))] {
		assert.NotEmpty(t, location, "Location should not be empty")
	}

	// Verify coordinate format (sample check)
	for _, coords := range coordinatePairs[:min(3, len(coordinatePairs))] {
		assert.Contains(t, coords, ",", "Coordinates should contain comma separator")
		assert.NotEmpty(t, coords, "Coordinates should not be empty")
	}

	t.Logf("Successfully extracted %d URLs, %d locations, %d coordinate pairs",
		len(extractedURLs), len(extractedLocations), len(coordinatePairs))
}

// TestScraperIntegration tests end-to-end scraper workflow with mocking
func TestScraperIntegration(t *testing.T) {
	t.Run("processes detail page with repository integration", func(t *testing.T) {
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
