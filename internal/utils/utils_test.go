package utils

import (
	"net/http"
	"reflect"
	"testing"
)

func TestFlattenHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    *http.Header
		expected map[string]string
	}{
		{"nil", nil, nil},
		{"single-value", func() *http.Header { h := http.Header{}; h.Set("Content-Type", "application/json"); return &h }(), map[string]string{"Content-Type": "application/json"}},
		{"multi-value", func() *http.Header {
			h := http.Header{}
			h.Add("Accept", "text/html")
			h.Add("Accept", "application/xml")
			return &h
		}(), map[string]string{"Accept": "text/html; application/xml"}},
		{"multiple-keys", func() *http.Header { h := http.Header{}; h.Set("Foo", "a"); h.Set("Bar", "b"); return &h }(), map[string]string{"Foo": "a", "Bar": "b"}},
	}

	for _, c := range tests {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := FlattenHeaders(c.input)
			if !reflect.DeepEqual(got, c.expected) {
				t.Fatalf("%s: got %v, expected %v", c.name, got, c.expected)
			}
		})
	}
}
