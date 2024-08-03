package app_test

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/wroge/sqlt"
	"github.com/wroge/vertical-slice-architecture/app"
)

type noLogTB struct{}

func (noLogTB) Helper()                         {}
func (noLogTB) Log(args ...any)                 {}
func (noLogTB) Logf(format string, args ...any) {}

func BenchGetBooks(b *testing.B, url string) {
	db, err := sql.Open("sqlite3", "file:test.db?cache=shared&mode=memory")
	if err != nil {
		log.Panicf("Failed to open database: %v", err)
	}

	defer db.Close()

	a := &app.App{
		Dialect:  "sqlite",
		Template: sqlt.New("db"),
		DB:       db,
		Logger:   log.New(os.Stdout, "Benchmark Book API - ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	_, api := humatest.New(noLogTB{})

	a.Init(api, &app.Options{})
	if err := a.FillFakeData(); err != nil {
		b.Fatal(err)
	}

	// warming up
	for range 100 {
		resp := api.Get(url)
		if resp.Code != 200 {
			b.Fatalf("Unexpected status code: %d", resp.Code)
		}
	}

	b.ResetTimer()

	// Benchmark loop
	for range b.N {
		resp := api.Get(url)
		if resp.Code != 200 {
			b.Fatalf("Unexpected status code: %d", resp.Code)
		}

		var output app.GetBooksOutputBody
		if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
			b.Fatal(err)
		}

		if len(output.Books) != 100 {
			b.Fatalf("Expected 100 books, got %d", len(output.Books))
		}
	}
}

func BenchmarkGetBooksSquirrel(b *testing.B) {
	BenchGetBooks(b, "http://localhost:8080/squirrel/books?limit=100")
}

func BenchmarkGetBooksSqlt(b *testing.B) {
	BenchGetBooks(b, "http://localhost:8080/sqlt/books?limit=100")
}
