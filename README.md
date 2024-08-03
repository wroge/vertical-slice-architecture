# Vertical Slice Architecture

This repository showcases how to build a Vertical Slice API that supports multiple SQL dialects using [huma](https://github.com/danielgtaylor/huma) and [sqlt](https://github.com/wroge/sqlt).

```go
// Run as local In-memory sqlite app
go run ./cmd/sqlite/main.go
// open: http://localhost:8080/docs


// Or run as postgres app with docker
docker run --name postgres -e POSTGRES_USER=user -e POSTGRES_PASSWORD=password -e POSTGRES_DB=db -p 5432:5432 -d postgres:16
go run ./cmd/postgres/main.go
// open: http://localhost:8080/docs

// stop and remove container:
docker stop postgres && docker rm postgres
```

## What is Vertical Slice Architecture?

- [https://www.jimmybogard.com/vertical-slice-architecture/](https://www.jimmybogard.com/vertical-slice-architecture/)
- [https://www.milanjovanovic.tech/blog/vertical-slice-architecture](https://www.milanjovanovic.tech/blog/vertical-slice-architecture)

> Minimize coupling between slices, and maximize coupling in a slice. (Jimmy Bogard)

## The Slices

- Query Books with sqlt: [app/get_books_sqlt.go](https://github.com/wroge/vertical-slice-architecture/blob/main/app/get_books_sqlt.go)
- Query Books with squirrel: [app/get_books_squirrel.go](https://github.com/wroge/vertical-slice-architecture/blob/main/app/get_books_squirrel.go)
- Insert Book with sqlt: [app/post_books_sqlt.go](https://github.com/wroge/vertical-slice-architecture/blob/main/app/post_books_sqlt.go)

## Highlights

- Huma is a great framework for OpenAPI generation and validation that works with any web framework.
- Without a repository layer, each query is designed perfectly for its specific use case, avoiding poor reuse.
- The SQL templates are a flexible and powerful tool that let you focus on business logic.

## Benchmarks

Of course, the standard/squirrel way is a little faster than sqlt. Look at both slices and decide if it’s worth using sqlt.

```
go test -bench . -benchmem ./app -benchtime=10s -count=2
goos: darwin
goarch: arm64
pkg: github.com/wroge/vertical-slice-architecture/app
BenchmarkGetBooksSquirrel-12                5048           2374140 ns/op          592942 B/op       4361 allocs/op
BenchmarkGetBooksSquirrel-12                4812           2357486 ns/op          589853 B/op       4329 allocs/op
BenchmarkGetBooksSqlt-12                    4976           2447785 ns/op          658196 B/op       4575 allocs/op
BenchmarkGetBooksSqlt-12                    4936           2480386 ns/op          667358 B/op       4632 allocs/op
PASS
ok      github.com/wroge/vertical-slice-architecture/app        52.586s
```

## Feedback

Take a look at the code and give me feedback. Thanks :)
