package model

import (
	"reflect"
)

// Restaurant stores information about a restaurant on Michelin Guide
type Restaurant struct {
	Name           string
	Address        string
	Price          string
	Type           string
	Longitude      string
	Latitude       string
	PhoneNumber    string
	Url            string
	WebsiteUrl     string
	Classification string
}

func (r Restaurant) ToSlice() reflect.Value {
	return reflect.ValueOf(r)
}

// Generate a slice of field names from a struct
func GenerateFieldNameSlice(class interface{}) []string {
	t := reflect.TypeOf(class)

	fields := make([]string, t.NumField())

	for i := range fields {
		fields[i] = t.Field(i).Name
	}

	return fields
}
