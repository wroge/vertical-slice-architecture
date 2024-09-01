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
BenchmarkGetBooksStandard100-12                     2396           5028297 ns/op          716218 B/op       6846 allocs/op
BenchmarkGetBooksStandardAlternative100-12          2557           4698100 ns/op          662655 B/op       3264 allocs/op
BenchmarkGetBooksSqlt100-12                         2346           5095556 ns/op          705397 B/op       6393 allocs/op
BenchmarkGetBooksSqltAlternative100-12              2562           4672304 ns/op          644436 B/op       2915 allocs/op
BenchmarkGetBooksStandard10-12                      3044           3937636 ns/op          104412 B/op       1376 allocs/op
BenchmarkGetBooksStandardAlternative10-12           3421           3476571 ns/op          100085 B/op        855 allocs/op
BenchmarkGetBooksSqlt10-12                          3025           3989344 ns/op           88241 B/op        919 allocs/op
BenchmarkGetBooksSqltAlternative10-12               3434           3536701 ns/op           76516 B/op        504 allocs/op
PASS
ok      github.com/wroge/vertical-slice-architecture/app        99.853s
```

## Feedback

Take a look at the code and give me feedback. Thanks :)
