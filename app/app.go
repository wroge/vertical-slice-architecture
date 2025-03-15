package app

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"

	"github.com/Masterminds/sprig/v3"
	"github.com/danielgtaylor/huma/v2"
	"github.com/wroge/sqlt"
)

//go:embed data.sql
var data string

type startKey struct{}

type Options struct {
	Port int  `help:"Port to listen on" short:"p" default:"8080"`
	Fill bool `help:"Fill with fake data" short:"f" default:"false"`
}

type App struct {
	Config sqlt.Config
	DB     *sql.DB
	Logger *slog.Logger
}

func (a *App) Init(api huma.API, options *Options) {
	a.Config.Templates = append(a.Config.Templates,
		sqlt.Funcs(sprig.TxtFuncMap()),
	)

	a.Config.Log = func(ctx context.Context, info sqlt.Info) {
		if info.Template == "data" {
			return
		}

		fmt.Println(info.SQL, info.Args, info.Location, info.Template, info.Err)
	}

	_, err := a.DB.Exec(`
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

		DROP INDEX IF EXISTS idx_books_title;
		CREATE INDEX idx_books_title ON books(LOWER(title));

		DROP INDEX IF EXISTS idx_authors_name;
		CREATE INDEX idx_authors_name ON authors(LOWER(name));
	`)
	if err != nil {
		a.Logger.Error(err.Error())
	}

	// add handlers here
	a.InsertBook(api)
	a.QueryBooks(api)

	if options.Fill {
		_, err = sqlt.Exec[any](a.Config, sqlt.Name("data"), sqlt.Parse(data)).Exec(context.Background(), a.DB, nil)
		if err != nil {
			a.Logger.Error("data already exists", "err", err)
		}
	}
}
