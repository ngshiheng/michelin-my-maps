package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitUnpack(t *testing.T) {
	cases := []struct {
		Str   string
		Price string
		Type  string
	}{
		{"145 - 195 EUR • Modern Cuisine, Classic Cuisin", "145 - 195 EUR", "Modern Cuisine, Classic Cuisin"},
		{"45 - 160 CHF • Italian Contemporary", "45 - 160 CHF", "Italian Contemporary"},
		{"75 - 115 EUR • Piedmontese, Creative", "75 - 115 EUR", "Piedmontese, Creative"},
	}

	for _, tt := range cases {
		t.Run("", func(t *testing.T) {
			got1, got2 := SplitUnpack(tt.Str, "•")
			assert.Equal(t, tt.Price, got1)
			assert.Equal(t, tt.Type, got2)
		})
	}

}
