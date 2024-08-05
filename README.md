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
BenchmarkGetBooksStandard100-12                     3975           3039698 ns/op          612565 B/op       4725 allocs/op
BenchmarkGetBooksStandardAlternative100-12          4400           2770984 ns/op          531379 B/op       1629 allocs/op
BenchmarkGetBooksSqlt100-12                         3750           3180468 ns/op          675222 B/op       4726 allocs/op
BenchmarkGetBooksSqltAlternative100-12              4351           2769914 ns/op          581451 B/op       1536 allocs/op
BenchmarkGetBooksStandard10-12                      4957           2412461 ns/op           77108 B/op        788 allocs/op
BenchmarkGetBooksStandardAlternative10-12           5785           2065329 ns/op           75206 B/op        481 allocs/op
BenchmarkGetBooksSqlt10-12                          4888           2455400 ns/op           86795 B/op        756 allocs/op
BenchmarkGetBooksSqltAlternative10-12               5737           2087308 ns/op           69845 B/op        356 allocs/op
PASS
ok      github.com/wroge/vertical-slice-architecture/app        98.567s
```

## Feedback

Take a look at the code and give me feedback. Thanks :)
