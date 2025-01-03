package app

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log/slog"
	"time"

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
	a.Config.TemplateOptions = append(a.Config.TemplateOptions,
		sqlt.Funcs(sprig.TxtFuncMap()),
	)

	a.Config.Start = func(runner *sqlt.Runner) {
		runner.Context = context.WithValue(runner.Context, startKey{}, time.Now())
	}

	a.Config.End = func(err error, runner *sqlt.Runner) {
		var attrs []slog.Attr

		if start, ok := runner.Context.Value(startKey{}).(time.Time); ok {
			attrs = append(attrs, slog.Duration("duration", time.Since(start)))
		}

		attrs = append(attrs,
			slog.String("sql", runner.SQL.String()),
			slog.Any("args", runner.Args),
			slog.String("location", fmt.Sprintf("[%s]", runner.Location)),
		)

		if err != nil {
			a.Logger.LogAttrs(runner.Context, slog.LevelError, err.Error(), attrs...)
		} else {
			a.Logger.LogAttrs(runner.Context, slog.LevelInfo, "log stmt", attrs...)
		}
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
	a.PostBooks(api)
	a.GetBooksSqltAlternative(api)

	if options.Fill {
		_, err = sqlt.Stmt[any](a.Config, sqlt.New("data"), sqlt.Parse(data)).Exec(context.Background(), a.DB, nil)
		if err != nil {
			a.Logger.Error("data already exists", "err", err)
		}
	}
}
