<h1 align="center"><strong>Michelin My Maps</strong></h1>

[![Continuos Integration](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/ci.yml/badge.svg)](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/ci.yml)
[![Semantic Release](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/release.yml/badge.svg)](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/release.yml)

-   [Context](#context)
-   [Disclaimer](#disclaimer)
-   [Content](#content)
-   [Usage](#usage)
    -   [Trip Feature](#trip-feature)
-   [Inspiration](#inspiration)
-   [Contributing](#contributing)

## Context

At the beginning of the automobile era, [Michelin](https://www.michelin.com/), a tire company, created a travel guide, including a restaurant guide.

Through the years, Michelin stars have become very prestigious due to their high standards and very strict anonymous testers. Michelin Stars are incredibly coveted. Gaining just one can change a chef's life; losing one, however, can change it as well.

The dataset is curated using [Go Colly](https://github.com/gocolly/colly).

[Read more...](https://jerrynsh.com/how-i-scraped-michelin-guide-using-golang/)

## Disclaimer

This software is only used for research purposes, users must abide by the relevant laws and regulations of their location, please do not use it for illegal purposes. The user shall bear all the consequences caused by illegal use.

## Content

The dataset contains a list of restaurants along with additional details (e.g. address, price, cuisine type, longitude, latitude, etc.) curated from the [MICHELIN Restaurants guide](https://guide.michelin.com/en/restaurants). The culinary distinctions of the restaurants included are:

-   3 Stars
-   2 Stars
-   1 Star
-   Bib Gourmand
-   Selected Restaurants

| Content | Link                                                                       | Description                    |
| :------ | :------------------------------------------------------------------------- | :----------------------------- |
| CSV     | [CSV](./data/michelin_my_maps.csv)                                         | Good'ol comma-separated values |
| Kaggle  | [Kaggle](https://www.kaggle.com/ngshiheng/michelin-guide-restaurants-2021) | Data science community         |

## Usage

### Trip Feature

Create personalized restaurant maps for your travels with the `trip` command. Filter restaurants by location, awards, cuisine, and more, then export to multiple map formats.

**Quick Start:**

```bash
# Create a trip with filters
mym trip create "Paris2025" -cities "Paris" -min-stars 1

# Export to interactive maps (GeoJSON, KML, HTML)
mym trip export "Paris2025"

# View your trips
mym trip list

# Delete a trip
mym trip delete "Paris2025"
```

**Available Filters:**
- `-cities` - Filter by cities/locations (comma-separated)
- `-distinctions` - Filter by award types (3 Stars, 2 Stars, 1 Star, Bib Gourmand)
- `-min-stars` - Minimum star rating (1, 2, or 3)
- `-cuisines` - Filter by cuisine types (comma-separated)
- `-green-star` - Only sustainable/green star restaurants

**Export Formats:**
- **GeoJSON** - For web maps and GIS applications
- **KML** - For Google Earth and offline mobile apps (Maps.me, OsmAnd)
- **HTML** - Standalone interactive map with Leaflet (works on any device)

**Example Use Cases:**

```bash
# Multi-city food tour
mym trip create "FrenchTour" -cities "Paris,Lyon,Marseille"

# 3-star bucket list
mym trip create "ThreeStarBucketList" -distinctions "3 Stars"

# Sustainable dining
mym trip create "GreenItaly" -cities "Italy" -green-star

# Budget-friendly options
mym trip create "BibGourmand" -cities "Tokyo" -distinctions "Bib Gourmand"
```

For detailed documentation, see [TRIP_FEATURE.md](./TRIP_FEATURE.md).

## Inspiration

Inspired by [this Reddit post](https://www.reddit.com/r/singapore/comments/pqnjd2/singapore_michelin_guide_2021_map/), my initial intention of creating this dataset is so that I can map all Michelin Guide Restaurants from all around the world on Google My Maps ([see an example](https://www.google.com/maps/d/edit?mid=1wSXxkPcNY50R78_T83tUZdZuYRk2L6jY&usp=sharing)).

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

1. Fork this
2. Create your feature branch (`git checkout -b feature/bar`)
3. Commit your changes (`git commit -am 'feat: add some bar'`, make sure that your commits are [semantic](https://www.conventionalcommits.org/en/v1.0.0/#summary))
4. Push to the branch (`git push origin feature/bar`)
5. Create a new Pull Request
