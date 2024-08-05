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
BenchmarkGetBooksStandard100-12                     4143           2901664 ns/op          604351 B/op       4637 allocs/op
BenchmarkGetBooksStandardAlternative100-12          4629           2606132 ns/op          525541 B/op       1541 allocs/op
BenchmarkGetBooksSqlt100-12                         3904           3074356 ns/op          664034 B/op       4609 allocs/op
BenchmarkGetBooksSqltAlternative100-12              4408           2682585 ns/op          569294 B/op       1418 allocs/op
BenchmarkGetBooksStandard10-12                      5091           2350112 ns/op           76217 B/op        773 allocs/op
BenchmarkGetBooksStandardAlternative10-12           5930           2016562 ns/op           74617 B/op        466 allocs/op
BenchmarkGetBooksSqlt10-12                          5001           2380299 ns/op           84597 B/op        713 allocs/op
BenchmarkGetBooksSqltAlternative10-12               5871           2031883 ns/op           67317 B/op        313 allocs/op
PASS
ok      github.com/wroge/vertical-slice-architecture/app        98.032s
```

## Feedback

Take a look at the code and give me feedback. Thanks :)
