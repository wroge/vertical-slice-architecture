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
	"github.com/go-sqlt/sqlt"
	"github.com/wroge/vertical-slice-architecture/app"
	_ "modernc.org/sqlite"
)

func main() {
	cli := humacli.New(func(hooks humacli.Hooks, options *app.Options) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		db, err := sql.Open("sqlite", "file:test.db?cache=shared&mode=memory")
		if err != nil {
			logger.Error(err.Error())
		}

		err = db.Ping()
		if err != nil {
			logger.Error(err.Error())
		}

		a := app.App{
			Config: sqlt.Sqlite(),
			DB:     db,
			Logger: logger,
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

			if err = server.ListenAndServe(); err != nil {
				logger.Error(err.Error())
			}
		})
	})

	cli.Run()
}
