# Vertical Slice Architecture

This repository showcases how to build a Vertical Slice API that supports multiple SQL dialects using [huma](https://github.com/danielgtaylor/huma) and [sqlt](https://github.com/wroge/sqlt).

```go
// Run as local In-memory sqlite app and fill with fake data
go run ./cmd/sqlite/main.go
// open: http://localhost:8080/docs


// Or run as postgres app with docker
docker run --name postgres -e POSTGRES_USER=user -e POSTGRES_PASSWORD=password -e POSTGRES_DB=db -p 5432:5432 -d postgres:16
go run ./cmd/postgres/main.go
// open: http://localhost:8080/docs

// stop and remove container:
docker stop postgres && docker rm postgres

// create new fake data
go run ./cmd/fake-data/main.go
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
BenchmarkGetBooksStandard100-12                     3992           3006379 ns/op          617744 B/op       4746 allocs/op
BenchmarkGetBooksStandardAlternative100-12          4498           2659571 ns/op          535497 B/op       1649 allocs/op
BenchmarkGetBooksSqlt100-12                         3826           3168712 ns/op          680449 B/op       4716 allocs/op
BenchmarkGetBooksSqltAlternative100-12              4323           2802906 ns/op          579289 B/op       1526 allocs/op
BenchmarkGetBooksStandard10-12                      4888           2428489 ns/op           81878 B/op        807 allocs/op
BenchmarkGetBooksStandardAlternative10-12           5832           2057900 ns/op           80319 B/op        501 allocs/op
BenchmarkGetBooksSqlt10-12                          4868           2460466 ns/op           85782 B/op        738 allocs/op
BenchmarkGetBooksSqltAlternative10-12               5760           2097406 ns/op           69789 B/op        344 allocs/op
PASS
ok      github.com/wroge/vertical-slice-architecture/app        98.544s
```

## Feedback

Take a look at the code and give me feedback. Thanks :)
