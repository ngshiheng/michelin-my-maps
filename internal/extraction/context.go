package extraction

import (
	"github.com/gocolly/colly/v2"
)

// RequestContext holds common context data extracted from colly requests.
type RequestContext struct {
	Location  string
	Latitude  string
	Longitude string
	Year      int
}

// ExtractContextData extracts common context values from a colly request context.
func ExtractContextData(ctx *colly.Context) RequestContext {
	var year int
	if v := ctx.GetAny("publishedYear"); v != nil {
		if y, ok := v.(int); ok {
			year = y
		}
	}

	return RequestContext{
		Location:  ctx.Get("location"),
		Latitude:  ctx.Get("latitude"),
		Longitude: ctx.Get("longitude"),
		Year:      year,
	}
}
