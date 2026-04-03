<h1 align="center"><strong>Michelin My Maps</strong></h1>

[![Continuos Integration](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/ci.yml/badge.svg)](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/ci.yml)
[![Semantic Release](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/release.yml/badge.svg)](https://github.com/ngshiheng/michelin-my-maps/actions/workflows/release.yml)

- [Context](#context)
- [Disclaimer](#disclaimer)
- [Content](#content)

## Context

At the beginning of the automobile era, [Michelin](https://www.michelin.com/), a tire company, created a travel guide, including a restaurant guide.

Through the years, Michelin stars have become very prestigious due to their high standards and very strict anonymous testers. Michelin Stars are incredibly coveted. Gaining just one can change a chef's life; losing one, however, can change it as well.

Built with [Go Colly](https://github.com/gocolly/colly).

[Read more...](https://jerrynsh.com/how-i-scraped-michelin-guide-using-golang/)

[...and more](https://jerrynsh.com/building-what-michelin-wouldnt-its-awards-history/)

## Disclaimer

This software is only used for research purposes, users must abide by the relevant laws and regulations of their location, please do not use it for illegal purposes. The user shall bear all the consequences caused by illegal use.

## Content

The dataset contains a list of restaurants along with additional details (e.g. address, price, cuisine type, longitude, latitude, etc.) curated from the [MICHELIN Restaurants guide](https://guide.michelin.com/en/restaurants). The culinary distinctions of the restaurants included are:

- 3 Stars
- 2 Stars
- 1 Star
- Bib Gourmand
- Selected Restaurants

| Content | Link                                                                       | Description                    |
| :------ | :------------------------------------------------------------------------- | :----------------------------- |
| CSV     | [CSV](./data/michelin_my_maps.csv)                                         | Good'ol comma-separated values |
| Kaggle  | [Kaggle](https://www.kaggle.com/ngshiheng/michelin-guide-restaurants-2021) | Data science community         |
