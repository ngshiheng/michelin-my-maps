package trip

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"os"
	"strconv"
	"strings"
)

// GeoJSON structures
type GeoJSON struct {
	Type     string        `json:"type"`
	Features []GeoFeature  `json:"features"`
}

type GeoFeature struct {
	Type       string                 `json:"type"`
	Geometry   GeoGeometry            `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

type GeoGeometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// KML structures
type KML struct {
	XMLName  xml.Name `xml:"kml"`
	XMLNS    string   `xml:"xmlns,attr"`
	Document Document `xml:"Document"`
}

type Document struct {
	Name      string      `xml:"name"`
	Placemarks []Placemark `xml:"Placemark"`
}

type Placemark struct {
	Name        string      `xml:"name"`
	Description string      `xml:"description"`
	Point       Point       `xml:"Point"`
}

type Point struct {
	Coordinates string `xml:"coordinates"`
}

// ExportGeoJSON exports restaurants to GeoJSON format
func ExportGeoJSON(restaurants []RestaurantWithAward, outputPath string) error {
	features := make([]GeoFeature, 0, len(restaurants))

	for _, r := range restaurants {
		lat, err := strconv.ParseFloat(r.Latitude, 64)
		if err != nil {
			continue
		}
		lng, err := strconv.ParseFloat(r.Longitude, 64)
		if err != nil {
			continue
		}

		feature := GeoFeature{
			Type: "Feature",
			Geometry: GeoGeometry{
				Type:        "Point",
				Coordinates: []float64{lng, lat}, // GeoJSON uses [lng, lat]
			},
			Properties: map[string]interface{}{
				"name":        r.Name,
				"description": r.Description,
				"address":     r.Address,
				"location":    r.Location,
				"cuisine":     r.Cuisine,
				"distinction": r.Award.Distinction,
				"green_star":  r.Award.GreenStar,
				"price":       r.Award.Price,
				"year":        r.Award.Year,
				"phone":       r.PhoneNumber,
				"website":     r.WebsiteURL,
				"guide_url":   r.URL,
			},
		}
		features = append(features, feature)
	}

	geoJSON := GeoJSON{
		Type:     "FeatureCollection",
		Features: features,
	}

	data, err := json.MarshalIndent(geoJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal GeoJSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write GeoJSON file: %w", err)
	}

	return nil
}

// ExportKML exports restaurants to KML format
func ExportKML(restaurants []RestaurantWithAward, outputPath, tripName string) error {
	placemarks := make([]Placemark, 0, len(restaurants))

	for _, r := range restaurants {
		lat, err := strconv.ParseFloat(r.Latitude, 64)
		if err != nil {
			continue
		}
		lng, err := strconv.ParseFloat(r.Longitude, 64)
		if err != nil {
			continue
		}

		greenStarBadge := ""
		if r.Award.GreenStar {
			greenStarBadge = " 🌱"
		}

		description := fmt.Sprintf(`<![CDATA[
<b>%s</b>%s<br/>
<i>%s</i><br/><br/>
%s<br/>
%s<br/><br/>
<b>Cuisine:</b> %s<br/>
<b>Price:</b> %s<br/>
<b>Phone:</b> %s<br/>
<a href="%s">Website</a> | <a href="%s">Michelin Guide</a>
]]>`,
			r.Award.Distinction,
			greenStarBadge,
			r.Cuisine,
			r.Description,
			r.Address,
			r.Cuisine,
			r.Award.Price,
			r.PhoneNumber,
			r.WebsiteURL,
			r.URL,
		)

		placemark := Placemark{
			Name:        r.Name,
			Description: description,
			Point: Point{
				Coordinates: fmt.Sprintf("%.6f,%.6f,0", lng, lat),
			},
		}
		placemarks = append(placemarks, placemark)
	}

	kml := KML{
		XMLNS: "http://www.opengis.net/kml/2.2",
		Document: Document{
			Name:      tripName,
			Placemarks: placemarks,
		},
	}

	data, err := xml.MarshalIndent(kml, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal KML: %w", err)
	}

	xmlData := append([]byte(xml.Header), data...)

	if err := os.WriteFile(outputPath, xmlData, 0644); err != nil {
		return fmt.Errorf("failed to write KML file: %w", err)
	}

	return nil
}

// ExportHTML exports restaurants to an interactive HTML map
func ExportHTML(restaurants []RestaurantWithAward, outputPath, tripName string) error {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.TripName}} - Michelin Restaurants</title>
    <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
    <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
    <style>
        body {
            margin: 0;
            padding: 0;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
        }
        #map {
            position: absolute;
            top: 0;
            bottom: 0;
            width: 100%;
        }
        .header {
            position: absolute;
            top: 10px;
            left: 50px;
            z-index: 1000;
            background: white;
            padding: 10px 20px;
            border-radius: 5px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.2);
        }
        .header h1 {
            margin: 0;
            font-size: 20px;
        }
        .popup-content {
            max-width: 300px;
        }
        .popup-title {
            font-size: 16px;
            font-weight: bold;
            margin-bottom: 5px;
        }
        .popup-distinction {
            color: #d4af37;
            font-weight: bold;
            margin-bottom: 5px;
        }
        .popup-cuisine {
            font-style: italic;
            color: #666;
            margin-bottom: 10px;
        }
        .popup-description {
            margin-bottom: 10px;
            font-size: 14px;
        }
        .popup-details {
            font-size: 12px;
            color: #666;
        }
        .popup-links {
            margin-top: 10px;
        }
        .popup-links a {
            margin-right: 10px;
            color: #0066cc;
            text-decoration: none;
        }
        .popup-links a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.TripName}}</h1>
        <p style="margin: 5px 0 0 0; color: #666;">{{.Count}} Michelin restaurants</p>
    </div>
    <div id="map"></div>
    <script>
        const restaurants = {{.RestaurantsJSON}};

        const map = L.map('map').setView([{{.CenterLat}}, {{.CenterLng}}], {{.Zoom}});

        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
            maxZoom: 19
        }).addTo(map);

        const starIcon = L.divIcon({
            className: 'custom-icon',
            html: '<div style="background-color: #d4af37; color: white; border-radius: 50%; width: 30px; height: 30px; display: flex; align-items: center; justify-content: center; font-weight: bold; border: 2px solid white; box-shadow: 0 2px 4px rgba(0,0,0,0.3);">⭐</div>',
            iconSize: [30, 30],
            iconAnchor: [15, 15]
        });

        restaurants.forEach(r => {
            const greenStar = r.green_star ? ' 🌱' : '';
            const popupContent = ` + "`" + `
                <div class="popup-content">
                    <div class="popup-title">${r.name}</div>
                    <div class="popup-distinction">${r.distinction}${greenStar}</div>
                    <div class="popup-cuisine">${r.cuisine}</div>
                    <div class="popup-description">${r.description}</div>
                    <div class="popup-details">
                        <strong>Address:</strong> ${r.address}<br/>
                        <strong>Price:</strong> ${r.price}<br/>
                        ${r.phone ? '<strong>Phone:</strong> ' + r.phone + '<br/>' : ''}
                    </div>
                    <div class="popup-links">
                        ${r.website ? '<a href="' + r.website + '" target="_blank">Website</a>' : ''}
                        <a href="${r.guide_url}" target="_blank">Michelin Guide</a>
                    </div>
                </div>
            ` + "`" + `;

            L.marker([r.lat, r.lng], { icon: starIcon })
                .addTo(map)
                .bindPopup(popupContent);
        });

        if (restaurants.length > 0) {
            const group = L.featureGroup(
                restaurants.map(r => L.marker([r.lat, r.lng]))
            );
            map.fitBounds(group.getBounds().pad(0.1));
        }
    </script>
</body>
</html>`

	// Prepare data for template
	type RestaurantData struct {
		Name        string  `json:"name"`
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
		Description string  `json:"description"`
		Address     string  `json:"address"`
		Cuisine     string  `json:"cuisine"`
		Distinction string  `json:"distinction"`
		GreenStar   bool    `json:"green_star"`
		Price       string  `json:"price"`
		Phone       string  `json:"phone"`
		Website     string  `json:"website"`
		GuideURL    string  `json:"guide_url"`
	}

	var restaurantData []RestaurantData
	var sumLat, sumLng float64
	validCount := 0

	for _, r := range restaurants {
		lat, err := strconv.ParseFloat(r.Latitude, 64)
		if err != nil {
			continue
		}
		lng, err := strconv.ParseFloat(r.Longitude, 64)
		if err != nil {
			continue
		}

		restaurantData = append(restaurantData, RestaurantData{
			Name:        strings.ReplaceAll(r.Name, `"`, `\"`),
			Lat:         lat,
			Lng:         lng,
			Description: strings.ReplaceAll(r.Description, `"`, `\"`),
			Address:     strings.ReplaceAll(r.Address, `"`, `\"`),
			Cuisine:     r.Cuisine,
			Distinction: r.Award.Distinction,
			GreenStar:   r.Award.GreenStar,
			Price:       r.Award.Price,
			Phone:       r.PhoneNumber,
			Website:     r.WebsiteURL,
			GuideURL:    r.URL,
		})

		sumLat += lat
		sumLng += lng
		validCount++
	}

	centerLat := 48.8566 // Default to Paris
	centerLng := 2.3522
	zoom := 6

	if validCount > 0 {
		centerLat = sumLat / float64(validCount)
		centerLng = sumLng / float64(validCount)
		zoom = 10
	}

	restaurantsJSON, err := json.Marshal(restaurantData)
	if err != nil {
		return fmt.Errorf("failed to marshal restaurant data: %w", err)
	}

	data := map[string]interface{}{
		"TripName":        tripName,
		"Count":           len(restaurantData),
		"RestaurantsJSON": template.JS(restaurantsJSON),
		"CenterLat":       centerLat,
		"CenterLng":       centerLng,
		"Zoom":            zoom,
	}

	t, err := template.New("map").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer f.Close()

	if err := t.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
