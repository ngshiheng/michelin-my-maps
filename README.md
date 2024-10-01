<h1 align="center"><strong>Michelin My Maps</strong></h1>

[![Continuos Integration](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/ci.yml/badge.svg)](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/ci.yml)
[![Semantic Release](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/release.yml/badge.svg)](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/release.yml)

- [Context](#context)
- [Disclaimer](#disclaimer)
- [Content](#content)
- [Inspiration](#inspiration)
- [Usage](#usage)
- [Development](#development)
  - [Selector](#selector)
  - [Testing](#testing)
  - [Caching](#caching)
- [Contributing](#contributing)

## Context

At the beginning of the automobile era, [Michelin](https://www.michelin.com/), a tire company, created a travel guide, including a restaurant guide.

Through the years, Michelin stars have become very prestigious due to their high standards and very strict anonymous testers. Michelin Stars are incredibly coveted. Gaining just one can change a chef's life; losing one, however, can change it as well.

The dataset is curated using [Go Colly](https://github.com/gocolly/colly).

[Read more...](https://jerrynsh.com/how-i-scraped-michelin-guide-using-golang/)

## Disclaimer

This software is only used for research purposes, users must abide by the relevant laws and regulations of their location, please do not use it for illegal purposes. The user shall bear all the consequences caused by illegal use.

## Content

The dataset contains a list of restaurants along with additional details (e.g. address, price range, cuisine type, longitude, latitude, etc.) curated from the [MICHELIN Restaurants guide](https://guide.michelin.com/en/restaurants). The culinary distinctions (i.e. the 'Award' column) of the restaurants included are:

-   3 Stars
-   2 Stars
-   1 Star
-   Bib Gourmand
-   Selected Restaurants

| Content | Link                                                                       | Description                    |
| :------ | :------------------------------------------------------------------------- | :----------------------------- |
| CSV     | [CSV](./data/michelin_my_maps.csv)                                         | Good'ol comma-separated values |
| Kaggle  | [Kaggle](https://www.kaggle.com/ngshiheng/michelin-guide-restaurants-2021) | Data science community         |

## Inspiration

Inspired by [this Reddit post](https://www.reddit.com/r/singapore/comments/pqnjd2/singapore_michelin_guide_2021_map/), my initial intention of creating this dataset is so that I can map all Michelin Guide Restaurants from all around the world on Google My Maps ([see an example](https://www.google.com/maps/d/edit?mid=1wSXxkPcNY50R78_T83tUZdZuYRk2L6jY&usp=sharing)).

## Usage

> **NOTE**
> Check out the [Makefile](./Makefile) or run `make help`.

To crawl, run:

```sh
make crawl # go run cmd/mym/mym.go
```

Alternatively, you can install this directly via `go install`:

```sh
go install github.com/ngshiheng/michelin-my-maps/v2/cmd/mym
rm michelin.db
mym -log debug
```

## Development

### Selector

As websites use JavaScript to dynamically generate content, the content may not be present in the initial HTML response. [Disabling JavaScript](https://developer.chrome.com/docs/devtools/javascript/disable/) can help you see the underlying HTML structure of the page and make it easier to identify the elements you want to scrape.

To extract relevant information from the site's HTML, we use XPath as our choice of selector language. You can make use of this [XPath cheat sheet](https://devhints.io/xpath).

### Testing

To run all tests locally, run:

```sh
make test # go test ./... -v -count=1
```

### Caching

Caching is enabled by default to avoid hammering the targeted site with too many unnecessary requests during development. After your first run, a [`cache/`](./cache/) folder (size of ~6GB) will be created. Your subsequent runs should be cached, they should take less than a minute to finish scraping the entire site.

To clear the cache, simply delete the [`cache/`](./cache/) folder.

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

1. Fork this
2. Create your feature branch (`git checkout -b feature/bar`)
3. Commit your changes (`git commit -am 'feat: add some bar'`, make sure that your commits are [semantic](https://www.conventionalcommits.org/en/v1.0.0/#summary))
4. Push to the branch (`git push origin feature/bar`)
5. Create a new Pull Request
