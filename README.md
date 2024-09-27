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
go test -bench . -benchmem ./... -benchtime=10s
goos: darwin
goarch: arm64
pkg: github.com/wroge/vertical-slice-architecture/app
cpu: Apple M3 Pro
BenchmarkGetBooksStandard100-12                     2439           4875252 ns/op          714530 B/op       6945 allocs/op
BenchmarkGetBooksStandardAlternative100-12          2706           4449036 ns/op          654608 B/op       3364 allocs/op
BenchmarkGetBooksSqlt100-12                         2439           4935801 ns/op          611411 B/op       4690 allocs/op
BenchmarkGetBooksSqltAlternative100-12              2709           4376639 ns/op          637359 B/op       3004 allocs/op
BenchmarkGetBooksStandard10-12                      3132           3832085 ns/op          103909 B/op       1386 allocs/op
BenchmarkGetBooksStandardAlternative10-12           3627           3280711 ns/op           99196 B/op        865 allocs/op
BenchmarkGetBooksSqlt10-12                          3079           3883390 ns/op           78904 B/op        750 allocs/op
BenchmarkGetBooksSqltAlternative10-12               3595           3324089 ns/op           77386 B/op        503 allocs/op
PASS
ok      github.com/wroge/vertical-slice-architecture/app        99.262s
```

## Feedback

Take a look at the code and give me feedback. Thanks :)
