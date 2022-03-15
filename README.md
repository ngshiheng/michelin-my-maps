<h1 align="center"><strong>Michelin My Maps</strong></h1>

## Context

At the beginning of the automobile era, [Michelin](https://www.michelin.com/), a tire company, created a travel guide, including a restaurant guide. Through the years, Michelin stars have become very prestigious due to their high standards and very strict anonymous testers. Michelin Stars are incredibly coveted. Gaining just one can change a chef's life; losing one, however, can change it as well.

## Content

This dataset contains a list of restaurants along with additional details (e.g. address, price range, cuisine type, longitude, latitude, etc.) curated from the [MICHELIN Restaurants guide](https://guide.michelin.com/en/restaurants). The culinary distinctions (i.e. 'Award' column in the dataset) of the restaurants included are:

-   3 Stars
-   2 Stars
-   1 Star
-   Bib Gourmand

The dataset is curated using [Go Colly](https://github.com/gocolly/colly).

| Content | Link                                                                       | Description                                |
| :------ | :------------------------------------------------------------------------- | :----------------------------------------- |
| CSV     | [CSV](./generated/michelin_my_maps.csv)                                    | Good'ol comma-separated values             |
| Kaggle  | [Kaggle](https://www.kaggle.com/ngshiheng/michelin-guide-restaurants-2021) | Data science community                     |
| Search  | [Polymer Search](https://app.polymersearch.com/jerrynsh/michelin_my_maps/) | For advanced search and data visualization |

## Usage

Check out the [Makefile](./Makefile) for basic usage.

To begin scraping, run:

```sh
make run # go run main.go
```

To build binary, run:

```sh
make build # go build -o bin/main .
```

The output of this command is the [csv file](./generated/michelin_my_maps.csv) created/updated inside [`generated/`](./generated/) folder.

## Development

### Selector

To extract relevant information from the site's HTML, we use XPath as our choice of selector language. You can make use of this [XPath cheat sheet](https://devhints.io/xpath).

### Testing

To run all tests locally, run:

```sh
make test # go test ./... -v -count=1
```

### Caching

Caching is enabled by default to avoid hammering the targeted site with too many unnecessary requests during development. After your first run, a [`cache/`](./cache/) folder (size of ~1.3GB) will be created. Your subsequent runs should be cached, they should take less than a minute to finish scraping the entire site.

To clear the cache, simply delete the [`cache/`](./cache/) folder.

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

### Steps

1. Fork this
2. Create your feature branch (`git checkout -b feature/bar`)
3. Commit your changes (`git commit -am 'feat: add some bar'`, make sure that your commits are [semantic](https://www.conventionalcommits.org/en/v1.0.0/#summary))
4. Push to the branch (`git push origin feature/bar`)
5. Create a new Pull Request

## Inspiration

Inspired by [this Reddit post](https://www.reddit.com/r/singapore/comments/pqnjd2/singapore_michelin_guide_2021_map/), my intention of creating this dataset is so that I can map all Michelin Guide Restaurants from all around the world on Google My Map.
