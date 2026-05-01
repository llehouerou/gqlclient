package decode

import (
	"reflect"
	"strings"
	"sync"

	"github.com/llehouerou/gqlclient/internal/reflectutil"
	"github.com/llehouerou/gqlclient/internal/tagparser"
	"github.com/llehouerou/gqlclient/types"
)

// fieldEntry is the precomputed match candidate for one struct field. The
// decoder walks a slice of these in declaration order, mirroring the
// original linear scan but without paying for reflect.Tag.Lookup or tag
// parsing on every JSON key.
//
// Per-call work is preserved only for fields whose type implements
// types.GraphQLType, since some implementations derive the name from the
// instance (see ActualNodes[T].GetGraphQLType in graphql_test.go).
type fieldEntry struct {
	fieldIndex int
	isScalar   bool

	// hasGraphQLType is true when the field's type implements
	// types.GraphQLType. Such fields resolve their match name from the
	// live value at lookup time.
	hasGraphQLType bool
	fieldType      reflect.Type

	// staticName is the precomputed match name from the graphql tag /
	// alias / struct field name. Empty means there is no static match
	// (fragments, graphql:"-" with empty key, etc.).
	staticName string
	// staticFold makes the static name compare case-insensitively. Used
	// for the untagged fallback that originally relied on
	// strings.EqualFold(field.Name, key).
	staticFold bool
	// hasStatic distinguishes "no static match at all" (fragment) from
	// "static match against empty string". Tags like graphql:"-" set
	// hasStatic=true with staticName="-", preserving prior behavior.
	hasStatic bool
}

// fieldLookupTable holds precomputed match candidates for one struct type.
type fieldLookupTable struct {
	entries []fieldEntry
}

// lookup walks the precomputed entries in declaration order and returns
// the first matching field. v is the parent struct value, used to resolve
// dynamic GraphQLType-derived names.
func (t *fieldLookupTable) lookup(
	v reflect.Value,
	name string,
) (idx int, scalar, ok bool) {
	for i := range t.entries {
		e := &t.entries[i]

		if e.hasGraphQLType {
			fv := v.Field(e.fieldIndex)
			if liveName, gtok := reflectutil.GetGraphQLType(
				fv, e.fieldType,
			); gtok {
				// liveName may be parameterized like "card(slug:$slug)"
				// or aliased like "x: card(...)" — the JSON response
				// key is just the bare field/alias name. Run it
				// through the same tag parser the legacy decoder used
				// so we match on FieldName/Alias, not the raw string.
				if keyHasGraphQLName(liveName, name) {
					return e.fieldIndex, e.isScalar, true
				}
				// GraphQLType produced a name that did not match — the
				// original code does not fall through to tag/fold for
				// this field, so skip it.
				continue
			}
			// GraphQLType returned no usable name (e.g. nil interface).
			// Fall through to the static matcher, matching original
			// hasGraphQLName behavior.
		}

		if !e.hasStatic {
			continue
		}
		if e.staticFold {
			if strings.EqualFold(e.staticName, name) {
				return e.fieldIndex, e.isScalar, true
			}
			continue
		}
		if e.staticName == name {
			return e.fieldIndex, e.isScalar, true
		}
	}
	return 0, false, false
}

// fieldLookupCache memoizes fieldLookupTable per reflect.Type. Reads are
// lock-free on the hot path; first-encounter builds use LoadOrStore so
// concurrent first-decodes of the same type race harmlessly.
var fieldLookupCache sync.Map // map[reflect.Type]*fieldLookupTable

// lookupFieldTable returns the cached field lookup table for struct type
// t, building it on first encounter.
func lookupFieldTable(t reflect.Type) *fieldLookupTable {
	if cached, ok := fieldLookupCache.Load(t); ok {
		return cached.(*fieldLookupTable)
	}
	tbl := buildFieldLookupTable(t)
	actual, _ := fieldLookupCache.LoadOrStore(t, tbl)
	return actual.(*fieldLookupTable)
}

// buildFieldLookupTable scans struct type t once, deriving a fieldEntry
// per exported field. Fragments are skipped entirely since the original
// keyHasGraphQLName always returns false for them.
func buildFieldLookupTable(t reflect.Type) *fieldLookupTable {
	tbl := &fieldLookupTable{
		entries: make([]fieldEntry, 0, t.NumField()),
	}

	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			// Unexported field — never a match target.
			continue
		}

		entry := fieldEntry{
			fieldIndex:     i,
			isScalar:       reflectutil.IsTrue(sf.Tag.Get(types.ScalarTag)),
			hasGraphQLType: reflectutil.ImplementsGraphQLType(sf.Type),
			fieldType:      sf.Type,
		}

		if tag, ok := sf.Tag.Lookup(types.GraphQLTag); ok {
			parsed, err := tagparser.ParseGraphQLTag(tag)
			if err == nil && parsed.IsFragment {
				// Fragments never match a regular JSON key. Skip them
				// entirely so the linear walk on hot paths is shorter.
				continue
			}
			if err == nil {
				name := parsed.FieldName
				if parsed.Alias != "" {
					name = parsed.Alias
				}
				entry.staticName = name
				entry.hasStatic = true
			}
		} else {
			// No tag: case-insensitive struct-field-name match.
			entry.staticName = sf.Name
			entry.staticFold = true
			entry.hasStatic = true
		}

		tbl.entries = append(tbl.entries, entry)
	}

	return tbl
}
