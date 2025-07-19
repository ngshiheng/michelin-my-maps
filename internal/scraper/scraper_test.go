package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gocolly/colly/v2"
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

	cfg := DefaultConfig()
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
		url := e.ChildAttr(restaurantDetailURLXPath, "href")
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

	// Verify exact counts based on actual HTML content
	assert.Equal(t, 24, len(extractedURLs), "Should extract exactly 24 restaurant URLs")
	assert.Equal(t, 24, len(extractedLocations), "Should extract exactly 24 restaurant locations")
	assert.Equal(t, 24, len(coordinatePairs), "Should extract exactly 24 coordinate pairs")

	// Verify specific URLs from actual HTML content
	expectedURLs := []string{
		"/en/oslo-region/oslo/restaurant/maaemo-1194933",
		"/en/rogaland/stavanger/restaurant/re-naa",
		"/en/stockholm-region/stockholm/restaurant/frantzen",
	}
	for i, expectedURL := range expectedURLs {
		assert.Equal(t, expectedURL, extractedURLs[i], "URL at index %d should match expected value", i)
	}

	// Verify specific locations from actual HTML content
	expectedLocations := []string{
		"Oslo, Norway",
		"Stavanger, Norway",
		"Stockholm, Sweden",
	}
	for i, expectedLocation := range expectedLocations {
		assert.Contains(t, extractedLocations[i], expectedLocation, "Location at index %d should contain expected value", i)
	}

	// Verify all URLs follow the expected pattern
	for i, url := range extractedURLs {
		assert.Contains(t, url, "/restaurant/", "URL at index %d should contain '/restaurant/': %s", i, url)
		assert.True(t, strings.HasPrefix(url, "/en/"), "URL at index %d should start with '/en/': %s", i, url)
		assert.NotEmpty(t, url, "URL at index %d should not be empty", i)
	}

	// Verify all locations are properly formatted
	for i, location := range extractedLocations {
		assert.NotEmpty(t, location, "Location at index %d should not be empty", i)
		// Note: Some locations like "Dubai" may not have country separators
		if !strings.Contains(location, ",") {
			t.Logf("Location at index %d has no country separator: %s", i, location)
		}
	}

	// Verify all coordinate pairs are valid
	for i, coords := range coordinatePairs {
		assert.Contains(t, coords, ",", "Coordinates at index %d should contain comma separator: %s", i, coords)
		assert.NotEmpty(t, coords, "Coordinates at index %d should not be empty", i)
		// Verify coordinate format (latitude,longitude)
		parts := strings.Split(coords, ",")
		assert.Len(t, parts, 2, "Coordinates at index %d should have exactly 2 parts: %s", i, coords)
		// Basic validation that they look like numbers
		for j, part := range parts {
			assert.NotEmpty(t, strings.TrimSpace(part), "Coordinate part %d at index %d should not be empty: %s", j, i, coords)
		}
	}

	t.Logf("Successfully extracted %d URLs, %d locations, %d coordinate pairs", len(extractedURLs), len(extractedLocations), len(coordinatePairs))
	t.Logf("Sample URLs: %v", extractedURLs[:min(3, len(extractedURLs))])
	t.Logf("Sample locations: %v", extractedLocations[:min(3, len(extractedLocations))])
	t.Logf("Sample coordinates: %v", coordinatePairs[:min(3, len(coordinatePairs))])
}

// TestScraperIntegration tests end-to-end scraper workflow with mocking
func TestScraperIntegration(t *testing.T) {
	t.Run("processes detail page with repository integration", func(t *testing.T) {
		htmlContent := loadTestHTML(t, "restaurant_detail.html")
		server := createTestServer(htmlContent)
		defer server.Close()

		mockRepo := &MockRepository{}
		mockRepo.On("UpsertRestaurantWithAward", context.Background(), mock.Anything).Return(nil)

		scraper, _ := New()

		// Test the detail page extraction workflow
		c := colly.NewCollector()
		c.OnXML(restaurantDetailXPath, func(e *colly.XMLElement) {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
