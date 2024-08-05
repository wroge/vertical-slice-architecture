package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/danielgtaylor/huma/v2/humacli"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wroge/sqlt"
	"github.com/wroge/vertical-slice-architecture/app"
)

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, options *app.Options) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		db, err := sql.Open("sqlite3", "file:test.db?cache=shared&mode=memory")
		if err != nil {
			logger.Error(err.Error())
		}

		err = db.Ping()
		if err != nil {
			logger.Error(err.Error())
		}

		a := app.App{
			Dialect:  app.Sqlite,
			Template: sqlt.New("db"),
			DB:       db,
			Logger:   logger,
		}

		router := http.NewServeMux()

		a.Init(humago.New(router, huma.DefaultConfig("Book API", "1.0.0")), options)

		server := &http.Server{
			Addr:         fmt.Sprintf(":%d", options.Port),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
			Handler:      router,
		}

		// Tell the CLI how to start your router.
		hooks.OnStart(func() {
			logger.Info("API started...")

			server.ListenAndServe()
		})
	})

	cli.Run()
}
