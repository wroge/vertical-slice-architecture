package app

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"log/slog"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/danielgtaylor/huma/v2"
	"github.com/wroge/sqlt"
)

//go:embed data.sql
var data string

type startKey struct{}

const (
	Postgres = "Postgres"
	Sqlite   = "Sqlite"
)

type Options struct {
	Port int  `help:"Port to listen on" short:"p" default:"8080"`
	Fill bool `help:"Fill with fake data" short:"f" default:"false"`
}

type App struct {
	Template *sqlt.Template
	DB       *sql.DB
	Logger   *slog.Logger
	Dialect  string
}

func (a *App) Init(api huma.API, options *Options) {
	a.Template.
		Funcs(sprig.TxtFuncMap()).
		Funcs(template.FuncMap{
			Postgres: func() bool {
				return a.Dialect == Postgres
			},
			Sqlite: func() bool {
				return a.Dialect == Sqlite
			},
		}).
		BeforeRun(func(op sqlt.Operation, r *sqlt.Runner) {
			r.Context = context.WithValue(r.Context, startKey{}, time.Now())
		}).
		AfterRun(func(err error, op sqlt.Operation, r *sqlt.Runner) error {
			dur := time.Since(r.Context.Value(startKey{}).(time.Time))

			if err != nil {
				// ignore sql.ErrNoRows
				if op == sqlt.FetchAllOperation && errors.Is(err, sql.ErrNoRows) {
					return nil
				}

				// apply error logging here
				a.Logger.Error(err.Error(), "template", r.Text.Name(), "duration", dur, "sql", r.SQL, "args", r.Args)

				return err
			}

			if r.Text.Name() != "data" {
				// apply normal logging here
				a.Logger.Info("query executed", "template", r.Text.Name(), "duration", dur, "sql", r.SQL, "args", r.Args)
			}

			return nil
		})

	_, err := a.Template.New("create").MustParse(`
		CREATE TABLE IF NOT EXISTS books (
			id TEXT PRIMARY KEY, 
			title TEXT NOT NULL, 
			number_of_pages INTEGER NOT NULL, 
			published_at DATE NOT NULL 
		);
		CREATE TABLE IF NOT EXISTS authors (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL
		);
		CREATE TABLE IF NOT EXISTS book_authors (
			book_id TEXT NOT NULL,
			author_id TEXT NOT NULL,
			PRIMARY KEY (book_id, author_id)
		);
		CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);
		CREATE INDEX IF NOT EXISTS idx_authors_name ON authors(name);
		CREATE INDEX IF NOT EXISTS idx_book_authors_book_id_author_id ON book_authors(book_id, author_id);
	`).Exec(context.Background(), a.DB, nil)
	if err != nil {
		a.Logger.Error(err.Error())
	}

	// add handlers here
	a.PostBooks(api)
	a.GetBooksSqlt(api)
	a.GetBooksSqltAlternative(api)
	a.GetBooksStandard(api)
	a.GetBooksStandardAlternative(api)

	_, err = a.Template.New("data").MustParse(data).Exec(context.Background(), a.DB, nil)
	if err != nil {
		a.Logger.Error(err.Error())
	}
}
