package crawler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateFieldValueSlice(t *testing.T) {
	type TestStruct struct {
		Field1 string
		Field2 string
	}

	cases := []struct {
		Got      interface{}
		Expected []string
	}{
		{
			TestStruct{
				"value1",
				"10",
			},
			[]string{"value1", "10"},
		},
		{
			Restaurant{
				Name:                  "My Restaurant",
				Address:               "123 Main St",
				Location:              "New York, NY",
				Price:                 "$",
				Cuisine:               "Italian",
				Longitude:             "-74.0059",
				Latitude:              "40.7128",
				PhoneNumber:           "212-555-1234",
				Url:                   "myrestaurant.com",
				WebsiteUrl:            "https://www.myrestaurant.com",
				Award:                 "Best Italian Restaurant",
				FacilitiesAndServices: "Outdoor seating, takeout, delivery",
			},
			[]string{
				"My Restaurant",
				"123 Main St",
				"New York, NY",
				"$",
				"Italian",
				"-74.0059",
				"40.7128",
				"212-555-1234",
				"myrestaurant.com",
				"https://www.myrestaurant.com",
				"Best Italian Restaurant",
				"Outdoor seating, takeout, delivery",
			},
		},
	}

	for _, tt := range cases {
		t.Run("test GenerateFieldValueSlice", func(t *testing.T) {
			got := GenerateFieldValueSlice(tt.Got)
			assert.Equal(t, tt.Expected, got)
		})
	}
}

func TestGenerateFieldNameSlice(t *testing.T) {
	type TestStruct struct {
		Field1 string
		Field2 int
		Field3 map[string]int
	}

	cases := []struct {
		Got      interface{}
		Expected []string
	}{
		{
			TestStruct{
				Field1: "value1",
				Field2: 10,
				Field3: map[string]int{"key1": 1, "key2": 2},
			},
			[]string{"Field1", "Field2", "Field3"},
		},
		{
			Restaurant{
				Name:                  "My Restaurant",
				Address:               "123 Main St",
				Location:              "New York, NY",
				Price:                 "$",
				Cuisine:               "Italian",
				Longitude:             "-74.0059",
				Latitude:              "40.7128",
				PhoneNumber:           "212-555-1234",
				Url:                   "myrestaurant.com",
				WebsiteUrl:            "https://www.myrestaurant.com",
				Award:                 "Best Italian Restaurant",
				FacilitiesAndServices: "Outdoor seating, takeout, delivery",
			},
			[]string{
				"Name",
				"Address",
				"Location",
				"Price",
				"Cuisine",
				"Longitude",
				"Latitude",
				"PhoneNumber",
				"Url",
				"WebsiteUrl",
				"Award",
				"FacilitiesAndServices",
			},
		},
	}

	for _, tt := range cases {
		t.Run("test TestGenerateFieldNameSlice", func(t *testing.T) {
			got := GenerateFieldNameSlice(tt.Got)
			assert.Equal(t, tt.Expected, got)
		})
	}
}
