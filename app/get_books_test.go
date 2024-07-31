package app_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/wroge/sqlt"
	"github.com/wroge/vertical-slice-architecture/app"
)

var (
	a   *app.App
	api humatest.TestAPI
)

func BenchGetBooks(b *testing.B, url string) {
	if api == nil {
		_, api = humatest.New(b)

		a.Init(api, &app.Options{})
		if err := a.FillFakeData(); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	// Benchmark loop
	for i := 0; i < b.N; i++ {
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

func BenchmarkGetBooksStandard(b *testing.B) {
	BenchGetBooks(b, "http://localhost:8080/standard/books?limit=100")
}

func BenchmarkGetBooksSqlt(b *testing.B) {
	BenchGetBooks(b, "http://localhost:8080/sqlt/books?limit=100")
}

func TestMain(m *testing.M) {
	db, err := sql.Open("sqlite3", "file:test.db?cache=shared&mode=memory")
	if err != nil {
		log.Panicf("Failed to open database: %v", err)
	}

	defer db.Close()

	log.SetOutput(io.Discard)

	a = &app.App{
		Dialect: "sqlite",
		Template: sqlt.New("db").HandleErr(func(err sqlt.Error) error {
			if errors.Is(err.Err, sql.ErrNoRows) {
				return nil
			}

			return err.Err
		}),
		DB:     db,
		Logger: log.New(os.Stdout, "Benchmark Book API - ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	os.Exit(m.Run())
}
