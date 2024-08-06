# Vertical Slice Architecture

This repository showcases how to build a Vertical Slice API that supports multiple SQL dialects using [huma](https://github.com/danielgtaylor/huma) and [sqlt](https://github.com/wroge/sqlt).

```go
// Run as local In-memory sqlite app and fill with fake data
go run ./cmd/sqlite/main.go --fill=true
// open: http://localhost:8080/docs


// Or run as postgres app with docker
docker run --name postgres -e POSTGRES_USER=user -e POSTGRES_PASSWORD=password -e POSTGRES_DB=db -p 5432:5432 -d postgres:16
go run ./cmd/postgres/main.go --fill=true
// open: http://localhost:8080/docs

// stop and remove container:
docker stop postgres && docker rm postgres
```

## What is Vertical Slice Architecture?

- [https://www.jimmybogard.com/vertical-slice-architecture/](https://www.jimmybogard.com/vertical-slice-architecture/)
- [https://www.milanjovanovic.tech/blog/vertical-slice-architecture](https://www.milanjovanovic.tech/blog/vertical-slice-architecture)

> Minimize coupling between slices, and maximize coupling in a slice. (Jimmy Bogard)

## The Slices

In the alternative slice, the list of books is directly converted into a JSON array in the database. In the regular version, two queries are executed to obtain the total number of books available before pagination.

- Query Books Standard: [app/get_books_standard.go](https://github.com/wroge/vertical-slice-architecture/blob/main/app/get_books_standard.go)
- Query Books Standard Alternative: [app/get_books_standard_alternative.go](https://github.com/wroge/vertical-slice-architecture/blob/main/app/get_books_standard_alternative.go)
- Query Books sqlt: [app/get_books_sqlt.go](https://github.com/wroge/vertical-slice-architecture/blob/main/app/get_books_sqlt.go)
- Query Books sqlt Alternative: [app/get_books_sqlt_alternative.go](https://github.com/wroge/vertical-slice-architecture/blob/main/app/get_books_sqlt_alternative.go)
- Insert Book with sqlt: [app/post_books_sqlt.go](https://github.com/wroge/vertical-slice-architecture/blob/main/app/post_books_sqlt.go)

## Highlights

- Huma is a great framework for OpenAPI generation and validation that works with any web framework.
- Without a repository layer, each query is designed perfectly for its specific use case, avoiding poor reuse.
- The SQL templates are a flexible and powerful tool that let you focus on business logic.

## Benchmarks

```
go test -bench . -benchmem ./app -benchtime=10s
goos: darwin
goarch: arm64
pkg: github.com/wroge/vertical-slice-architecture/app
BenchmarkGetBooksStandard100-12                     4030           2932113 ns/op          608947 B/op       4691 allocs/op
BenchmarkGetBooksStandardAlternative100-12          4611           2579027 ns/op          528377 B/op       1595 allocs/op
BenchmarkGetBooksSqlt100-12                         3969           3069970 ns/op          668091 B/op       4663 allocs/op
BenchmarkGetBooksSqltAlternative100-12              4484           2647152 ns/op          572257 B/op       1472 allocs/op
BenchmarkGetBooksStandard10-12                      5152           2377542 ns/op           76283 B/op        778 allocs/op
BenchmarkGetBooksStandardAlternative10-12           5941           2012358 ns/op           74698 B/op        471 allocs/op
BenchmarkGetBooksSqlt10-12                          5013           2381804 ns/op           83069 B/op        717 allocs/op
BenchmarkGetBooksSqltAlternative10-12               5916           2035125 ns/op           67016 B/op        318 allocs/op
PASS
ok      github.com/wroge/vertical-slice-architecture/app        98.291s
```

## Feedback

Take a look at the code and give me feedback. Thanks :)
