package fragments_test

import (
	"reflect"
	"testing"

	"github.com/llehouerou/gqlclient/internal/fragments"
)

func TestFromField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		tag          string
		wantTypename string
		wantOK       bool
	}{
		{
			name:         "valid fragment with typename",
			tag:          `graphql:"... on Droid"`,
			wantTypename: "Droid",
			wantOK:       true,
		},
		{
			name:         "valid fragment without typename",
			tag:          `graphql:"..."`,
			wantTypename: "",
			wantOK:       true,
		},
		{
			name:         "regular field",
			tag:          `graphql:"name"`,
			wantTypename: "",
			wantOK:       false,
		},
		{
			name:         "field with arguments",
			tag:          `graphql:"height(unit: METER)"`,
			wantTypename: "",
			wantOK:       false,
		},
		{
			name:         "field with alias",
			tag:          `graphql:"node1: node(id: $id)"`,
			wantTypename: "",
			wantOK:       false,
		},
		{
			name:         "no graphql tag",
			tag:          `json:"name"`,
			wantTypename: "",
			wantOK:       false,
		},
		{
			name:         "empty tag",
			tag:          ``,
			wantTypename: "",
			wantOK:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			field := reflect.StructField{
				Name: "TestField",
				Type: reflect.TypeOf(""),
				Tag:  reflect.StructTag(tt.tag),
			}
			typename, ok := fragments.FromField(field)
			if ok != tt.wantOK {
				t.Errorf("FromField() ok = %v, want %v", ok, tt.wantOK)
			}
			if typename != tt.wantTypename {
				t.Errorf(
					"FromField() typename = %q, want %q",
					typename,
					tt.wantTypename,
				)
			}
		})
	}
}

func TestFromTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		tagValue     string
		wantTypename string
		wantOK       bool
	}{
		{
			name:         "fragment with typename",
			tagValue:     "... on Droid",
			wantTypename: "Droid",
			wantOK:       true,
		},
		{
			name:         "fragment with long typename",
			tagValue:     "... on SolanaTokenTransferAuthorizationRequest",
			wantTypename: "SolanaTokenTransferAuthorizationRequest",
			wantOK:       true,
		},
		{
			name:         "fragment without typename",
			tagValue:     "...",
			wantTypename: "",
			wantOK:       true,
		},
		{
			name:         "fragment with extra spaces",
			tagValue:     "...  on  Droid",
			wantTypename: "Droid",
			wantOK:       true,
		},
		{
			name:         "regular field name",
			tagValue:     "name",
			wantTypename: "",
			wantOK:       false,
		},
		{
			name:         "field with arguments",
			tagValue:     "height(unit: METER)",
			wantTypename: "",
			wantOK:       false,
		},
		{
			name:         "field with alias",
			tagValue:     "node1: node(id: $id)",
			wantTypename: "",
			wantOK:       false,
		},
		{
			name:         "empty string",
			tagValue:     "",
			wantTypename: "",
			wantOK:       false,
		},
		{
			name:         "skip field marker",
			tagValue:     "-",
			wantTypename: "",
			wantOK:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			typename, ok := fragments.FromTag(tt.tagValue)
			if ok != tt.wantOK {
				t.Errorf("FromTag(%q) ok = %v, want %v", tt.tagValue, ok, tt.wantOK)
			}
			if typename != tt.wantTypename {
				t.Errorf(
					"FromTag(%q) typename = %q, want %q",
					tt.tagValue,
					typename,
					tt.wantTypename,
				)
			}
		})
	}
}
