package app

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/brianvoe/gofakeit"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/wroge/sqlt"
	"golang.org/x/exp/rand"
)

type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"8080"`
}

type App struct {
	Template *sqlt.Template[any]
	DB       *sql.DB
	Dialect  string
	Logger   *log.Logger
}

func (a *App) Init(api huma.API, options *Options) {
	a.Template.Funcs(sprig.TxtFuncMap()).Funcs(template.FuncMap{
		"Dialect": func() string {
			return a.Dialect
		},
	})

	_, err := a.Template.New("create").MustParse(`
		CREATE TABLE IF NOT EXISTS books (
			id TEXT PRIMARY KEY, -- uuid
			title TEXT NOT NULL, 
			number_of_pages INTEGER NOT NULL,
			published_at DATE NOT NULL
		);
		CREATE TABLE IF NOT EXISTS authors (
			id TEXT PRIMARY KEY, -- uuid
			name TEXT UNIQUE NOT NULL
		);
		CREATE TABLE IF NOT EXISTS book_authors (
			book_id TEXT NOT NULL,
			author_id TEXT NOT NULL,
			PRIMARY KEY (book_id, author_id)
		);
	`).Exec(context.Background(), a.DB, nil)
	if err != nil {
		a.Logger.Panic(err)
	}

	// add handlers here
	a.PostBooks(api)
	a.GetBooksSqlt(api)
	a.GetBooksStandard(api)
}

func (a *App) FillFakeData() error {
	// Seed the random number generator
	gofakeit.Seed(0)
	var (
		buffer  bytes.Buffer
		books   [1000]uuid.UUID
		authors [100]uuid.UUID
	)

	// Generate 1000 books
	for i := 0; i < 1000; i++ {
		books[i] = uuid.New()
		title := gofakeit.Sentence(3)         // Generates a fake book title
		numberOfPages := rand.Intn(900) + 100 // Random number between 100 and 999
		publishedAt := randomDate().Format("2006-01-02")

		buffer.WriteString(fmt.Sprintf("INSERT INTO books (id, title, number_of_pages, published_at) VALUES ('%s', '%s', %d, '%s');\n",
			books[i], title, numberOfPages, publishedAt))
	}

	// Generate 100 authors
	for i := 0; i < 100; i++ {
		authors[i] = uuid.New()
		name := gofakeit.Name() // Generates a fake author name

		buffer.WriteString(fmt.Sprintf("INSERT INTO authors (id, name) VALUES ('%s', '%s');\n", authors[i], name))
	}

	// Generate book_authors relationships
	for i := 0; i < 1000; i++ {
		index := rand.Intn(100)
		author := authors[index]
		buffer.WriteString(fmt.Sprintf("INSERT INTO book_authors (book_id, author_id) VALUES ('%s', '%s');\n", books[i], author))

		next := rand.Intn(100)
		nextAuthor := authors[next]

		if index == next {
			continue
		}

		buffer.WriteString(fmt.Sprintf("INSERT INTO book_authors (book_id, author_id) VALUES ('%s', '%s');\n", books[i], nextAuthor))
	}

	_, err := a.DB.Exec(buffer.String())

	return err
}

// randomDate generates a random date between 1950 and 2022
func randomDate() time.Time {
	min := time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC).Unix()
	sec := rand.Int63n(max-min) + min
	return time.Unix(sec, 0)
}
