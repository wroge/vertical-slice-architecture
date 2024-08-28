package app_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	_ "modernc.org/sqlite"

	"github.com/wroge/sqlt"
	"github.com/wroge/vertical-slice-architecture/app"
)

type noLogTB struct{}

func (noLogTB) Helper()                         {}
func (noLogTB) Log(args ...any)                 {}
func (noLogTB) Logf(format string, args ...any) {}

var api humatest.TestAPI

func BenchGetBooks(b *testing.B, alt string, limit uint64) {
	for range b.N {
		resp := api.Get(fmt.Sprintf("http://localhost:8080/%s/books?limit=%d&search=a&offet=10&sort=number_of_pages&direction=desc", alt, limit))
		if resp.Code != 200 {
			b.Fatalf("Unexpected status code: %d", resp.Code)
		}

		var output app.GetBooksOutputBody
		if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetBooksStandard100(b *testing.B) {
	BenchGetBooks(b, "standard", 100)
}

func BenchmarkGetBooksStandardAlternative100(b *testing.B) {
	BenchGetBooks(b, "standard_alternative", 100)
}

func BenchmarkGetBooksSqlt100(b *testing.B) {
	BenchGetBooks(b, "sqlt", 100)
}

func BenchmarkGetBooksSqltAlternative100(b *testing.B) {
	BenchGetBooks(b, "sqlt_alternative", 100)
}

func BenchmarkGetBooksStandard10(b *testing.B) {
	BenchGetBooks(b, "standard", 10)
}

func BenchmarkGetBooksStandardAlternative10(b *testing.B) {
	BenchGetBooks(b, "standard_alternative", 10)
}

func BenchmarkGetBooksSqlt10(b *testing.B) {
	BenchGetBooks(b, "sqlt", 10)
}

func BenchmarkGetBooksSqltAlternative10(b *testing.B) {
	BenchGetBooks(b, "sqlt_alternative", 10)
}

func TestMain(m *testing.M) {
	db, err := sql.Open("sqlite", "file:test.db?cache=shared&mode=memory")
	if err != nil {
		log.Panicf("Failed to open database: %v", err)
	}

	defer db.Close()

	a := &app.App{
		Dialect:  "sqlite",
		Template: sqlt.New("db"),
		DB:       db,
		Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelError,
		})),
	}

	_, api = humatest.New(noLogTB{})

	a.Init(api, &app.Options{
		Fill: true,
	})

	m.Run()
}
