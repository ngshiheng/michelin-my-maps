package trip

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Trip represents a travel plan with restaurant filters
type Trip struct {
	Name        string    `json:"name"`
	Cities      []string  `json:"cities,omitempty"`      // Filter by cities/locations
	Distinctions []string `json:"distinctions,omitempty"` // Filter by award types
	MinStars    int       `json:"min_stars,omitempty"`   // Minimum star rating (0-3)
	MaxPrice    string    `json:"max_price,omitempty"`   // Maximum price range
	Cuisines    []string  `json:"cuisines,omitempty"`    // Filter by cuisine types
	GreenStarOnly bool    `json:"green_star_only,omitempty"` // Only green star restaurants
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Manager handles trip storage and retrieval
type Manager struct {
	tripsFile string
}

// NewManager creates a new trip manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".mym")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &Manager{
		tripsFile: filepath.Join(configDir, "trips.json"),
	}, nil
}

// loadTrips loads all trips from the JSON file
func (m *Manager) loadTrips() (map[string]*Trip, error) {
	trips := make(map[string]*Trip)

	data, err := os.ReadFile(m.tripsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return trips, nil
		}
		return nil, fmt.Errorf("failed to read trips file: %w", err)
	}

	if len(data) == 0 {
		return trips, nil
	}

	if err := json.Unmarshal(data, &trips); err != nil {
		return nil, fmt.Errorf("failed to parse trips file: %w", err)
	}

	return trips, nil
}

// saveTrips saves all trips to the JSON file
func (m *Manager) saveTrips(trips map[string]*Trip) error {
	data, err := json.MarshalIndent(trips, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal trips: %w", err)
	}

	if err := os.WriteFile(m.tripsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write trips file: %w", err)
	}

	return nil
}

// Create creates a new trip
func (m *Manager) Create(trip *Trip) error {
	trips, err := m.loadTrips()
	if err != nil {
		return err
	}

	if _, exists := trips[trip.Name]; exists {
		return fmt.Errorf("trip %q already exists", trip.Name)
	}

	now := time.Now().UTC()
	trip.CreatedAt = now
	trip.UpdatedAt = now

	trips[trip.Name] = trip
	return m.saveTrips(trips)
}

// Get retrieves a trip by name
func (m *Manager) Get(name string) (*Trip, error) {
	trips, err := m.loadTrips()
	if err != nil {
		return nil, err
	}

	trip, exists := trips[name]
	if !exists {
		return nil, fmt.Errorf("trip %q not found", name)
	}

	return trip, nil
}

// List returns all trips
func (m *Manager) List() ([]*Trip, error) {
	tripsMap, err := m.loadTrips()
	if err != nil {
		return nil, err
	}

	trips := make([]*Trip, 0, len(tripsMap))
	for _, trip := range tripsMap {
		trips = append(trips, trip)
	}

	return trips, nil
}

// Delete removes a trip
func (m *Manager) Delete(name string) error {
	trips, err := m.loadTrips()
	if err != nil {
		return err
	}

	if _, exists := trips[name]; !exists {
		return fmt.Errorf("trip %q not found", name)
	}

	delete(trips, name)
	return m.saveTrips(trips)
}

// Update updates an existing trip
func (m *Manager) Update(trip *Trip) error {
	trips, err := m.loadTrips()
	if err != nil {
		return err
	}

	existing, exists := trips[trip.Name]
	if !exists {
		return fmt.Errorf("trip %q not found", trip.Name)
	}

	trip.CreatedAt = existing.CreatedAt
	trip.UpdatedAt = time.Now().UTC()

	trips[trip.Name] = trip
	return m.saveTrips(trips)
}
