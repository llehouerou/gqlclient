package fragments_test

import (
	"reflect"
	"testing"

	"github.com/llehouerou/gqlclient/internal/fragments"
)

func TestIsStructField(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want bool
	}{
		{
			name: "valid fragment with typename",
			tag:  `graphql:"... on Droid"`,
			want: true,
		},
		{
			name: "valid fragment without typename",
			tag:  `graphql:"..."`,
			want: true,
		},
		{
			name: "regular field",
			tag:  `graphql:"name"`,
			want: false,
		},
		{
			name: "field with arguments",
			tag:  `graphql:"height(unit: METER)"`,
			want: false,
		},
		{
			name: "field with alias",
			tag:  `graphql:"node1: node(id: $id)"`,
			want: false,
		},
		{
			name: "no graphql tag",
			tag:  `json:"name"`,
			want: false,
		},
		{
			name: "empty tag",
			tag:  ``,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a struct field with the given tag
			field := reflect.StructField{
				Name: "TestField",
				Type: reflect.TypeOf(""),
				Tag:  reflect.StructTag(tt.tag),
			}
			got := fragments.IsStructField(field)
			if got != tt.want {
				t.Errorf("IsStructField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTag(t *testing.T) {
	tests := []struct {
		name     string
		tagValue string
		want     bool
	}{
		{
			name:     "fragment with typename",
			tagValue: "... on Droid",
			want:     true,
		},
		{
			name:     "fragment without typename",
			tagValue: "...",
			want:     true,
		},
		{
			name:     "fragment with extra spaces",
			tagValue: "...  on  Droid",
			want:     true,
		},
		{
			name:     "regular field name",
			tagValue: "name",
			want:     false,
		},
		{
			name:     "field with arguments",
			tagValue: "height(unit: METER)",
			want:     false,
		},
		{
			name:     "field with alias",
			tagValue: "node1: node(id: $id)",
			want:     false,
		},
		{
			name:     "empty string",
			tagValue: "",
			want:     false,
		},
		{
			name:     "skip field marker",
			tagValue: "-",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fragments.IsTag(tt.tagValue)
			if got != tt.want {
				t.Errorf("IsTag(%q) = %v, want %v", tt.tagValue, got, tt.want)
			}
		})
	}
}

func TestExtractTypename(t *testing.T) {
	tests := []struct {
		name     string
		tagValue string
		want     string
	}{
		{
			name:     "fragment with typename",
			tagValue: "... on Droid",
			want:     "Droid",
		},
		{
			name:     "fragment with long typename",
			tagValue: "... on SolanaTokenTransferAuthorizationRequest",
			want:     "SolanaTokenTransferAuthorizationRequest",
		},
		{
			name:     "fragment without typename",
			tagValue: "...",
			want:     "",
		},
		{
			name:     "fragment with extra spaces",
			tagValue: "...  on  Droid",
			want:     "Droid",
		},
		{
			name:     "regular field name",
			tagValue: "name",
			want:     "",
		},
		{
			name:     "field with arguments",
			tagValue: "height(unit: METER)",
			want:     "",
		},
		{
			name:     "field with alias",
			tagValue: "node1: node(id: $id)",
			want:     "",
		},
		{
			name:     "empty string",
			tagValue: "",
			want:     "",
		},
		{
			name:     "skip field marker",
			tagValue: "-",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fragments.ExtractTypename(tt.tagValue)
			if got != tt.want {
				t.Errorf(
					"ExtractTypename(%q) = %q, want %q",
					tt.tagValue,
					got,
					tt.want,
				)
			}
		})
	}
}
