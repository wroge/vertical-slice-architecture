package app_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	_ "github.com/mattn/go-sqlite3"

	"github.com/wroge/sqlt"
	"github.com/wroge/vertical-slice-architecture/app"
)

type noLogTB struct{}

func (noLogTB) Helper()                         {}
func (noLogTB) Log(args ...any)                 {}
func (noLogTB) Logf(format string, args ...any) {}

var (
	db  *sql.DB
	a   *app.App
	api humatest.TestAPI
)

func BenchGetBooks(b *testing.B, alt string) {
	for range b.N {
		resp := api.Get(fmt.Sprintf("http://localhost:8080/%s/books?limit=100&search=e&offet=10&sort=number_of_pages&direction=desc", alt))
		if resp.Code != 200 {
			b.Fatalf("Unexpected status code: %d", resp.Code)
		}

		var output app.GetBooksOutputBody
		if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetBooksStandard(b *testing.B) {
	BenchGetBooks(b, "standard")
}

func BenchmarkGetBooksAlternative(b *testing.B) {
	BenchGetBooks(b, "standard_alternative")
}

func BenchmarkGetBooksSqlt(b *testing.B) {
	BenchGetBooks(b, "sqlt")
}

func BenchmarkGetBooksSqltAlternative(b *testing.B) {
	BenchGetBooks(b, "sqlt_alternative")
}

func TestMain(m *testing.M) {
	var err error

	db, err = sql.Open("sqlite3", "file:test.db?cache=shared&mode=memory")
	if err != nil {
		log.Panicf("Failed to open database: %v", err)
	}

	defer db.Close()

	a = &app.App{
		Dialect:  "sqlite",
		Template: sqlt.New("db"),
		DB:       db,
		Logger:   log.New(os.Stdout, "Benchmark Book API - ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	_, api = humatest.New(noLogTB{})

	a.Init(api, &app.Options{})

	if err := a.FillFakeData(); err != nil {
		log.Fatal(err)
	}

	m.Run()
}
