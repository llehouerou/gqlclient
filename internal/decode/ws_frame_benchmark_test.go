package decode_test

import (
	"testing"
	"time"

	"github.com/llehouerou/gqlclient/internal/decode"
)

// wsOffer mirrors a representative typed payload from a WS subscription —
// 12 scalar/struct fields, a nested struct, a slice of small structs, and a
// time.Time field that exercises the json.Unmarshaler fallback path. Field
// names use explicit graphql tags so the benchmark covers the same hot path
// production consumers hit.
type wsOffer struct {
	ID        string    `graphql:"id"`
	Maker     string    `graphql:"maker"`
	Taker     *string   `graphql:"taker"`
	Price     float64   `graphql:"price"`
	Quantity  int64     `graphql:"quantity"`
	CreatedAt time.Time `graphql:"createdAt"`
	UpdatedAt time.Time `graphql:"updatedAt"`
	Status    string    `graphql:"status"`
	Currency  string    `graphql:"currency"`
	Type      string    `graphql:"type"`
	Metadata  struct {
		Source string `graphql:"source"`
		Region string `graphql:"region"`
	} `graphql:"metadata"`
	Items []struct {
		ID    string  `graphql:"id"`
		Price float64 `graphql:"price"`
	} `graphql:"items"`
}

// wsOfferPayload is sized close to the production p50 (~336 bytes) and
// includes nested + slice content so the benchmark exercises array decoding
// and time.Time UnmarshalJSON.
var wsOfferPayload = []byte(`{` +
	`"id":"0x123abc","maker":"0xdeadbeef","taker":"0xcafebabe",` +
	`"price":1.234567,"quantity":1000,` +
	`"createdAt":"2026-04-30T12:00:00Z","updatedAt":"2026-04-30T12:01:00Z",` +
	`"status":"OPEN","currency":"USDC","type":"BID",` +
	`"metadata":{"source":"WS","region":"us-east-1"},` +
	`"items":[{"id":"a","price":1.1},{"id":"b","price":2.2},{"id":"c","price":3.3}]` +
	`}`)

func BenchmarkUnmarshalGraphQL_WSOffer(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(wsOfferPayload)))
	for b.Loop() {
		var got wsOffer
		if err := decode.UnmarshalGraphQL(wsOfferPayload, &got); err != nil {
			b.Fatal(err)
		}
	}
}
