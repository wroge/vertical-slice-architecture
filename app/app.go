package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/wroge/sqlt"
)

type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"8080"`
}

type App struct {
	Template *sqlt.Template
	DB       *sql.DB
	Logger   *log.Logger
}

func (a *App) Run() {
	a.Template.Funcs(sprig.TxtFuncMap())

	create := a.Template.New("create").MustParse(`
		CREATE TABLE IF NOT EXISTS books (
			id TEXT PRIMARY KEY, 
			title TEXT NOT NULL, 
			number_of_pages INTEGER NOT NULL, 
			published_at {{ if eq Dialect "postgres" }}TIMESTAMPTZ{{ else }}DATE{{ end }} NOT NULL
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
	`)

	_, err := create.Exec(context.Background(), a.DB, nil)
	if err != nil {
		a.Logger.Panic(err)
	}

	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		router := http.NewServeMux()

		api := humago.New(router, huma.DefaultConfig("Book API", "1.0.0"))

		// add handlers here
		a.PostBooks(api)
		a.GetBooks(api)

		server := http.Server{
			Addr:         fmt.Sprintf(":%d", options.Port),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
			ErrorLog:     a.Logger,
			Handler:      router,
		}

		// Tell the CLI how to start your router.
		hooks.OnStart(func() {
			a.Logger.Print("Book API started...")

			server.ListenAndServe()
		})
	})

	cli.Run()
}
