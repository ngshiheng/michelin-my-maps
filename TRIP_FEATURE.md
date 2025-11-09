# Trip Feature - Dynamic Restaurant Maps

The new `trip` command allows you to create personalized restaurant maps for your travels. This feature filters the Michelin restaurant database based on your preferences and exports maps in multiple formats.

## Quick Start

### 1. Create a Trip

Create a trip with custom filters:

```bash
# Example: Paris trip with 1+ star restaurants
mym trip create "Paris2025" -cities "Paris" -min-stars 1

# Example: Multi-city French tour
mym trip create "FrenchTour" -cities "Paris,Lyon,Marseille" -cuisines "French"

# Example: 3-star bucket list
mym trip create "ThreeStarBucketList" -distinctions "3 Stars"

# Example: Green star sustainable restaurants in Italy
mym trip create "SustainableItaly" -cities "Italy" -green-star

# Example: Bib Gourmand budget-friendly options
mym trip create "BudgetFriendly" -cities "Tokyo,Osaka" -distinctions "Bib Gourmand"
```

### 2. List Your Trips

View all saved trips:

```bash
mym trip list
```

Output example:
```
Found 2 trip(s):

  Paris2025
    Cities: Paris
    Min Stars: 1
    Created: 2025-11-09

  FrenchTour
    Cities: Paris, Lyon, Marseille
    Cuisines: French
    Created: 2025-11-09
```

### 3. Export to Maps

Export your trip to interactive map formats:

```bash
# Export to all formats (GeoJSON, KML, HTML)
mym trip export "Paris2025"

# Export to specific format
mym trip export "Paris2025" -format html -output ./maps

# Export for offline use
mym trip export "Paris2025" -format kml
```

### 4. Delete a Trip

Remove a trip you no longer need:

```bash
mym trip delete "Paris2025"
```

## Filter Options

### Cities
Filter restaurants by location (partial match):
```bash
-cities "Paris,Lyon,Tokyo"
```

### Distinctions
Filter by award type:
- `3 Stars` - Three Michelin stars
- `2 Stars` - Two Michelin stars
- `1 Star` - One Michelin star
- `Bib Gourmand` - Good food at moderate prices
- `Selected Restaurants` - Recommended restaurants

```bash
-distinctions "3 Stars,2 Stars"
```

### Minimum Stars
Shortcut to filter by minimum star rating (1, 2, or 3):
```bash
-min-stars 2  # Include 2 and 3 star restaurants
```

### Cuisines
Filter by cuisine type:
```bash
-cuisines "French,Italian,Japanese"
```

### Green Star Only
Filter for sustainable/eco-friendly restaurants:
```bash
-green-star
```

## Export Formats

### GeoJSON (.geojson)
- Standard geographic data format
- Works with most mapping applications
- Easy to integrate with web maps (Mapbox, Leaflet, etc.)
- Contains full restaurant details as properties

**Use for:**
- Custom web map integration
- GIS analysis
- Modern mapping applications

### KML (.kml)
- Google Earth compatible
- Includes rich descriptions with formatting
- Works offline in many apps

**Use for:**
- Google Earth visualization
- Offline mobile apps (Maps.me, Osmand)
- Sharing with others

### HTML (.html)
- Standalone interactive map
- No internet required after first load (map tiles need internet)
- Beautiful Leaflet-based interface
- Works on any device with a browser

**Use for:**
- Quick viewing on any device
- Sharing via email/Dropbox/etc.
- Offline travel reference (save the HTML file)

## Example Workflows

### Planning a Paris Trip
```bash
# Create trip for Paris Michelin stars
mym trip create "Paris2025" -cities "Paris" -min-stars 1

# Export to HTML for easy viewing
mym trip export "Paris2025" -format html

# Open Paris2025.html in your browser
# The map shows all 1+ star restaurants with details
```

### Food Tour Across France
```bash
# Create multi-city tour
mym trip create "FranceTour" -cities "Paris,Lyon,Bordeaux,Marseille"

# Export to KML for offline use
mym trip export "FranceTour" -format kml

# Import FranceTour.kml into Google Maps or Maps.me
```

### 3-Star Michelin Bucket List
```bash
# Create global 3-star list
mym trip create "ThreeStarBucketList" -distinctions "3 Stars"

# Export to all formats
mym trip export "ThreeStarBucketList"

# Result: ThreeStarBucketList.geojson, .kml, and .html
```

### Sustainable Dining
```bash
# Green star restaurants in Europe
mym trip create "GreenEurope" -cities "France,Italy,Spain,Germany" -green-star

# Export to HTML
mym trip export "GreenEurope" -format html
```

### Budget-Friendly Options
```bash
# Bib Gourmand in Japan
mym trip create "JapanBibGourmand" -cities "Tokyo,Kyoto,Osaka" -distinctions "Bib Gourmand"

# Export and view
mym trip export "JapanBibGourmand" -format html
```

## Tips

1. **City Names**: Use the way Michelin spells them. Common formats:
   - Cities: "Paris", "Tokyo", "New York"
   - Regions: "Tuscany", "Provence"
   - Countries: "Italy", "France", "Japan"

2. **Multiple Filters**: Combine filters for precise results:
   ```bash
   mym trip create "LuxuryParis" -cities "Paris" -min-stars 2 -cuisines "French"
   ```

3. **HTML Maps**: The HTML files are self-contained. Save them to your phone/tablet for offline reference (note: map tiles require internet).

4. **GeoJSON**: Use with tools like:
   - [geojson.io](https://geojson.io) - View/edit online
   - QGIS - Advanced GIS analysis
   - Your own web maps with Leaflet/Mapbox

5. **KML**: Works great with:
   - Google Earth (desktop/mobile)
   - Google Maps (import to "Your Places")
   - Maps.me (offline mobile maps)
   - OsmAnd (offline mobile maps)

## Technical Details

- Trips are stored in `~/.mym/trips.json`
- Filters query the latest award year for each restaurant
- All coordinates are in WGS84 (standard lat/lng)
- HTML maps use OpenStreetMap tiles and Leaflet.js
- GeoJSON follows RFC 7946 specification
- KML follows OGC KML 2.2 specification

## Future Enhancements

Potential features to add:
- Route optimization (visit restaurants along your travel path)
- Date range filters (e.g., "restaurants awarded 2023-2025")
- Price range filters
- Distance-based filtering (within X km of a point)
- Export to Apple Maps
- Generate shareable URLs
- Mobile app integration
