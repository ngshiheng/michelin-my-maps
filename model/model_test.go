package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateFieldValueSlice(t *testing.T) {
	cases := []struct {
		Got      Restaurant
		Expected []string
	}{
		{
			Restaurant{
				"The Table Kevin Fehlin",
				"Shanghaiallee 15, Hamburg, 20457, Germany",
				"Hamburg",
				"230",
				"230",
				"EUR",
				"Creative",
				"Air conditioning,American Express credit card,Car park",
				"53.5426229",
				"10.0029797",
				"+494022867422",
				"https://guide.michelin.com/en/hamburg-region/hamburg/restaurant/the-table-kevin-fehling",
				"https://thetable-hamburg.de/",
				"Three MICHELIN Stars: Exceptional cuisine, worth a special journey!",
			},
			[]string{
				"The Table Kevin Fehlin",
				"Shanghaiallee 15, Hamburg, 20457, Germany",
				"Hamburg",
				"230",
				"230",
				"EUR",
				"Creative",
				"Air conditioning,American Express credit card,Car park",
				"53.5426229",
				"10.0029797",
				"+494022867422",
				"https://guide.michelin.com/en/hamburg-region/hamburg/restaurant/the-table-kevin-fehling",
				"https://thetable-hamburg.de/",
				"Three MICHELIN Stars: Exceptional cuisine, worth a special journey!",
			},
		},
	}

	for _, tt := range cases {
		t.Run("test split GenerateFieldValueSlice", func(t *testing.T) {
			got := GenerateFieldValueSlice(tt.Got)
			assert.Equal(t, tt.Expected, got)
		})
	}
}
