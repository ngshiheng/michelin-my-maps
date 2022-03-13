package model

import (
	"reflect"
)

// Restaurant stores information about a restaurant on Michelin Guide
type Restaurant struct {
	Name        string
	Address     string
	Location    string
	MinPrice    string
	MaxPrice    string
	Currency    string
	Type        string
	Longitude   string
	Latitude    string
	PhoneNumber string
	Url         string
	WebsiteUrl  string
	Award       string
}

// Generate field values slice from struct
func GenerateFieldValueSlice(class interface{}) []string {
	v := reflect.Indirect(reflect.ValueOf(class))

	var values = []string{}

	for i := 0; i < v.NumField(); i++ {
		values = append(values, v.Field(i).String())
	}

	return values
}

// Generate field names slice from struct
func GenerateFieldNameSlice(class interface{}) []string {
	t := reflect.TypeOf(class)

	fields := make([]string, t.NumField())

	for i := 0; i < len(fields); i++ {
		fields[i] = t.Field(i).Name
	}

	return fields
}
