gqlclient
=========

## About This Project

This project is a fork of [`github.com/hasura/go-graphql-client`](https://github.com/hasura/go-graphql-client), which itself was originally forked from [`github.com/shurcooL/graphql`](https://github.com/shurcooL/graphql).

### Key Changes from Upstream

This fork has diverged significantly from the original project:

- **Removed WebSocket subscription support**: The subscription client and all related WebSocket functionality have been removed to simplify the codebase and reduce dependencies
- **Removed examples**: All example code (graphqldev, realworld, subscription examples) has been removed
- **Module rename**: Changed from `github.com/hasura/go-graphql-client` to `github.com/llehouerou/gqlclient` for a shorter, more convenient import path
- **Modernization**: Updated to Go 1.25+ with modern Go idioms and tooling

Due to the extent of these changes, this project warrants its own module name and independent development path.

### What This Library Provides

Package `gqlclient` provides a simple, reflection-based GraphQL client implementation for Go. It focuses on query and mutation operations via HTTP, with a clean API for constructing GraphQL queries from Go structs.

- [gqlclient](#gqlclient)
	- [About This Project](#about-this-project)
		- [Key Changes from Upstream](#key-changes-from-upstream)
		- [What This Library Provides](#what-this-library-provides)
	- [Installation](#installation)
	- [Usage](#usage)
		- [Authentication](#authentication)
		- [Simple Query](#simple-query)
		- [Arguments and Variables](#arguments-and-variables)
		- [Custom scalar tag](#custom-scalar-tag)
		- [Skip GraphQL field](#skip-graphql-field)
		- [Inline Fragments](#inline-fragments)
		- [Specify GraphQL type name](#specify-graphql-type-name)
		- [Mutations](#mutations)
			- [Mutations Without Fields](#mutations-without-fields)
		- [Options](#options)
		- [Execute pre-built query](#execute-pre-built-query)
		- [Raw bytes response](#raw-bytes-response)
		- [Multiple mutations with ordered map](#multiple-mutations-with-ordered-map)
		- [Debugging and Unit test](#debugging-and-unit-test)
	- [Directories](#directories)
	- [References](#references)
	- [License](#license)
  
## Installation

`gqlclient` requires Go version 1.25 or later.

```bash
go get github.com/llehouerou/gqlclient
```

## Usage

Construct a GraphQL client, specifying the GraphQL server URL. Then, you can use it to make GraphQL queries and mutations.

```Go
client := graphql.NewClient("https://example.com/graphql", nil)
// Use client...
```

### Authentication

Some GraphQL servers may require authentication. The `graphql` package does not directly handle authentication. Instead, when creating a new client, you're expected to pass an `http.Client` that performs authentication. The easiest and recommended way to do this is to use the [`golang.org/x/oauth2`](https://golang.org/x/oauth2) package. You'll need an OAuth token with the right scopes. Then:

```Go
import "golang.org/x/oauth2"

func main() {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GRAPHQL_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	client := graphql.NewClient("https://example.com/graphql", httpClient)
	// Use client...
```

### Simple Query

To make a GraphQL query, you need to define a corresponding Go type. Variable names must be upper case, see [here](https://github.com/llehouerou/gqlclient/blob/master/README.md#specify-graphql-type-name)

For example, to make the following GraphQL query:

```GraphQL
query {
	me {
		name
	}
}
```

You can define this variable:

```Go
var query struct {
	Me struct {
		Name string
	}
}
```

Then call `client.Query`, passing a pointer to it:

```Go
err := client.Query(context.Background(), &query, nil)
if err != nil {
	// Handle error.
}
fmt.Println(query.Me.Name)

// Output: Luke Skywalker
```

### Arguments and Variables

Often, you'll want to specify arguments on some fields. You can use the `graphql` struct field tag for this.

For example, to make the following GraphQL query:

```GraphQL
{
	human(id: "1000") {
		name
		height(unit: METER)
	}
}
```

You can define this variable:

```Go
var q struct {
	Human struct {
		Name   string
		Height float64 `graphql:"height(unit: METER)"`
	} `graphql:"human(id: \"1000\")"`
}
```

Then call `client.Query`:

```Go
err := client.Query(context.Background(), &q, nil)
if err != nil {
	// Handle error.
}
fmt.Println(q.Human.Name)
fmt.Println(q.Human.Height)

// Output:
// Luke Skywalker
// 1.72
```

However, that'll only work if the arguments are constant and known in advance. Otherwise, you will need to make use of variables. Replace the constants in the struct field tag with variable names:

```Go
var q struct {
	Human struct {
		Name   string
		Height float64 `graphql:"height(unit: $unit)"`
	} `graphql:"human(id: $id)"`
}
```

Then, define a `variables` map with their values:

```Go
variables := map[string]interface{}{
	"id":   graphql.ID(id),
	"unit": starwars.LengthUnit("METER"),
}
```

Finally, call `client.Query` providing `variables`:

```Go
err := client.Query(context.Background(), &q, variables)
if err != nil {
	// Handle error.
}
```

### Custom scalar tag

Because the generator reflects recursively struct objects, it can't know if the struct is a custom scalar such as JSON. To avoid expansion of the field during query generation, let's add the tag `scalar:"true"` to the custom scalar. If the scalar implements the JSON decoder interface, it will be automatically decoded.

```Go
struct {
	Viewer struct {
		ID         interface{}
		Login      string
		CreatedAt  time.Time
		DatabaseID int
	}
}

// Output:
// {
//   viewer {
//	   id
//		 login
//		 createdAt
//		 databaseId
//   }	
// }

struct {
	Viewer struct {
		ID         interface{}
		Login      string
		CreatedAt  time.Time
		DatabaseID int
	} `scalar:"true"`
}

// Output
// { viewer }
```

### Skip GraphQL field

```go
struct {
  Viewer struct {
		ID         interface{} `graphql:"-"`
		Login      string
		CreatedAt  time.Time `graphql:"-"`
		DatabaseID int
  }
}

// Output
// {viewer{login,databaseId}}
```

### Inline Fragments

Some GraphQL queries contain inline fragments. You can use the `graphql` struct field tag to express them.

For example, to make the following GraphQL query:

```GraphQL
{
	hero(episode: "JEDI") {
		name
		... on Droid {
			primaryFunction
		}
		... on Human {
			height
		}
	}
}
```

You can define this variable:

```Go
var q struct {
	Hero struct {
		Name  string
		Droid struct {
			PrimaryFunction string
		} `graphql:"... on Droid"`
		Human struct {
			Height float64
		} `graphql:"... on Human"`
	} `graphql:"hero(episode: \"JEDI\")"`
}
```

Alternatively, you can define the struct types corresponding to inline fragments, and use them as embedded fields in your query:

```Go
type (
	DroidFragment struct {
		PrimaryFunction string
	}
	HumanFragment struct {
		Height float64
	}
)

var q struct {
	Hero struct {
		Name          string
		DroidFragment `graphql:"... on Droid"`
		HumanFragment `graphql:"... on Human"`
	} `graphql:"hero(episode: \"JEDI\")"`
}
```

Then call `client.Query`:

```Go
err := client.Query(context.Background(), &q, nil)
if err != nil {
	// Handle error.
}
fmt.Println(q.Hero.Name)
fmt.Println(q.Hero.PrimaryFunction)
fmt.Println(q.Hero.Height)

// Output:
// R2-D2
// Astromech
// 0
```

### Specify GraphQL type name

The GraphQL type is automatically inferred from Go type by reflection. However, it's cumbersome in some use cases, e.g lowercase names. In Go, a type name with a first lowercase letter is considered private. If we need to reuse it for other packages, there are 2 approaches: type alias or implement `GetGraphQLType` method.

```go
type UserReviewInput struct {
	Review string
	UserID string
}

// type alias
type user_review_input UserReviewInput
// or implement GetGraphQLType method
func (u UserReviewInput) GetGraphQLType() string { return "user_review_input" }

variables := map[string]interface{}{
  "input": UserReviewInput{}
}

//query arguments without GetGraphQLType() defined
//($input: UserReviewInput!)
//query arguments with GetGraphQLType() defined:w
//($input: user_review_input!)
```

### Mutations

Mutations often require information that you can only find out by performing a query first. Let's suppose you've already done that.

For example, to make the following GraphQL mutation:

```GraphQL
mutation($ep: Episode!, $review: ReviewInput!) {
	createReview(episode: $ep, review: $review) {
		stars
		commentary
	}
}
variables {
	"ep": "JEDI",
	"review": {
		"stars": 5,
		"commentary": "This is a great movie!"
	}
}
```

You can define:

```Go
var m struct {
	CreateReview struct {
		Stars      int
		Commentary string
	} `graphql:"createReview(episode: $ep, review: $review)"`
}
variables := map[string]interface{}{
	"ep": starwars.Episode("JEDI"),
	"review": starwars.ReviewInput{
		Stars:      5,
		Commentary: "This is a great movie!",
	},
}
```

Then call `client.Mutate`:

```Go
err := client.Mutate(context.Background(), &m, variables)
if err != nil {
	// Handle error.
}
fmt.Printf("Created a %v star review: %v\n", m.CreateReview.Stars, m.CreateReview.Commentary)

// Output:
// Created a 5 star review: This is a great movie!
```

#### Mutations Without Fields

Sometimes, you don't need any fields returned from a mutation. Doing that is easy.

For example, to make the following GraphQL mutation:

```GraphQL
mutation($ep: Episode!, $review: ReviewInput!) {
	createReview(episode: $ep, review: $review)
}
variables {
	"ep": "JEDI",
	"review": {
		"stars": 5,
		"commentary": "This is a great movie!"
	}
}
```

You can define:

```Go
var m struct {
	CreateReview string `graphql:"createReview(episode: $ep, review: $review)"`
}
variables := map[string]interface{}{
	"ep": starwars.Episode("JEDI"),
	"review": starwars.ReviewInput{
		Stars:      5,
		Commentary: "This is a great movie!",
	},
}
```

Then call `client.Mutate`:

```Go
err := client.Mutate(context.Background(), &m, variables)
if err != nil {
	// Handle error.
}
fmt.Printf("Created a review: %s.\n", m.CreateReview)

// Output:
// Created a review: .
```

### Options

There are extensible parts in the GraphQL query that we sometimes use. They are optional so that we shouldn't required them in the method. To make it flexible, we can abstract these options as optional arguments that follow this interface.

```go
type Option interface {
	Type() OptionType
	String() string
}

client.Query(ctx context.Context, q interface{}, variables map[string]interface{}, options ...Option) error
```

Currently we support 2 option types: `operation_name` and `operation_directive`. The operation name option is built-in because it is unique. We can use the option directly with `OperationName`

```go
// query MyQuery {
//	...
// }
client.Query(ctx, &q, variables, graphql.OperationName("MyQuery"))
```

In contrast, operation directive is various and customizable on different GraphQL servers. There isn't any built-in directive in the library. You need to define yourself. For example:

```go
// define @cached directive for Hasura queries
// https://hasura.io/docs/latest/graphql/cloud/response-caching.html#enable-caching
type cachedDirective struct {
	ttl int
}

func (cd cachedDirective) Type() OptionType {
	// operation_directive
	return graphql.OptionTypeOperationDirective
}

func (cd cachedDirective) String() string {
	if cd.ttl <= 0 {
		return "@cached"
	}
	return fmt.Sprintf("@cached(ttl: %d)", cd.ttl)
}

// query MyQuery @cached {
//	...
// }
client.Query(ctx, &q, variables, graphql.OperationName("MyQuery"), cachedDirective{})
```

### Execute pre-built query

The `Exec` function allows you to executing pre-built queries. While using reflection to build queries is convenient as you get some resemblance of type safety, it gets very cumbersome when you need to create queries semi-dynamically. For instance, imagine you are building a CLI tool to query data from a graphql endpoint and you want users to be able to narrow down the query by passing cli flags or something.

```Go
// filters would be built dynamically somehow from the command line flags
filters := []string{
   `fieldA: {subfieldA: {_eq: "a"}}`,
   `fieldB: {_eq: "b"}`,
   ...
}

query := "query{something(where: {" + strings.Join(filters, ", ") + "}){id}}"
res := struct {
	Somethings []Something
}{}

if err := client.Exec(ctx, query, &res, map[string]any{}); err != nil {
	panic(err)
}
```

If you prefer decoding JSON yourself, use `ExecRaw` instead.

```Go
query := `query{something(where: { foo: { _eq: "bar" }}){id}}`
var res struct {
	Somethings []Something `json:"something"`
}

raw, err := client.ExecRaw(ctx, query, map[string]any{}) 
if err != nil {
	panic(err)
}

err = json.Unmarshal(raw, &res)
```

### Raw bytes response

If you want to decode JSON response yourself, or the default `UnmarshalGraphQL` function isn't ideal for your use case, you can use the `*Raw` methods:

```Go
func (c *Client) QueryRaw(ctx context.Context, q interface{}, variables map[string]interface{}) ([]byte, error)

func (c *Client) MutateRaw(ctx context.Context, q interface{}, variables map[string]interface{}) ([]byte, error)
```

### Multiple mutations with ordered map

You might need to make multiple mutations in single query. It's not very convenient with structs
so you can use ordered map `[][2]interface{}` instead.

For example, to make the following GraphQL mutation:

```GraphQL
mutation($login1: String!, $login2: String!, $login3: String!) {
	createUser(login: $login1) { login }
	createUser(login: $login2) { login }
	createUser(login: $login3) { login }
}
variables {
	"login1": "grihabor",
	"login2": "diman",
	"login3": "indigo"
}
```

You can define:

```Go
type CreateUser struct {
	Login string
}
m := [][2]interface{}{
	{"createUser(login: $login1)", &CreateUser{}},
	{"createUser(login: $login2)", &CreateUser{}},
	{"createUser(login: $login3)", &CreateUser{}},
}
variables := map[string]interface{}{
	"login1": "grihabor",
	"login2": "diman",
	"login3": "indigo",
}
```

### Debugging and Unit test

Enable debug mode with the `WithDebug` function. If the request is failed, the request and response information will be included in `extensions[].internal` property.

```json
{
	"errors": [
		{
			"message":"Field 'user' is missing required arguments: login",
			"extensions": {
				"internal": {
					"request": {
						"body":"{\"query\":\"{user{name}}\"}",
						"headers": {
							"Content-Type": ["application/json"]
						}
					},
					"response": {
						"body":"{\"errors\": [{\"message\": \"Field 'user' is missing required arguments: login\",\"locations\": [{\"line\": 7,\"column\": 3}]}]}",
						"headers": {
							"Content-Type": ["application/json"]
						}
					}
				}
			},
			"locations": [
				{
					"line":7,
					"column":3
				}
			]
		}
	]
}
```

Because the GraphQL query string is generated in runtime using reflection, it isn't really safe. To assure the GraphQL query is expected, it's necessary to write some unit test for query construction.

```go
// ConstructQuery builds GraphQL query string from struct and variables
func ConstructQuery(v interface{}, variables map[string]interface{}, options ...Option) (string, error)

// ConstructMutation builds GraphQL mutation string from struct and variables
func ConstructMutation(v interface{}, variables map[string]interface{}, options ...Option) (string, error)

// UnmarshalGraphQL parses the JSON-encoded GraphQL response data and stores
// the result in the GraphQL query data structure pointed to by v.
func UnmarshalGraphQL(data []byte, v interface{}) error
```

## Directories

| Path                                                                           | Synopsis                                                                                                        |
|--------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------|
| [ident](https://pkg.go.dev/github.com/llehouerou/gqlclient/ident)             | Package ident provides functions for parsing and converting identifier names between various naming convention. |
| [types](https://pkg.go.dev/github.com/llehouerou/gqlclient/types)             | Package types provides GraphQL type interfaces and constants.                                                   |

## References

- Original project: [github.com/shurcooL/graphql](https://github.com/shurcooL/graphql)
- Upstream fork: [github.com/hasura/go-graphql-client](https://github.com/hasura/go-graphql-client)
- GraphQL specification: [https://graphql.org/](https://graphql.org/)

## License

[MIT License](LICENSE) - See LICENSE file for full text

Original work Copyright (c) 2017 Dmitri Shuralyov
Modified work Copyright (c) 2020 Hasura
Modified work Copyright (c) 2025 Laurent Le Houerou
