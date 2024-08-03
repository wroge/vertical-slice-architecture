package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
		logger := log.New(os.Stdout, "Sqlite Book API - ", log.Ldate|log.Ltime|log.Lshortfile)

		db, err := sql.Open("sqlite3", "file:test.db?cache=shared&mode=memory")
		if err != nil {
			logger.Panic(err)
		}

		err = db.Ping()
		if err != nil {
			logger.Fatal("Cannot connect to database", err)
		}

		a := app.App{
			Dialect: "sqlite",
			Template: sqlt.New("db").
				AfterRun(func(err error, name string, r *sqlt.Runner) error {
					if err != nil {
						// ignore sql.ErrNoRows
						if errors.Is(err, sql.ErrNoRows) {
							return nil
						}

						// apply error logging here
						fmt.Println(err, name, strings.Join(strings.Fields(r.SQL.String()), " "))

						return err
					}

					// apply normal logging here
					fmt.Println(name, strings.Join(strings.Fields(r.SQL.String()), " "))

					return err
				}),
			DB:     db,
			Logger: logger,
		}

		router := http.NewServeMux()

		a.Init(humago.New(router, huma.DefaultConfig("Book API", "1.0.0")), options)
		if err = a.FillFakeData(); err != nil {
			logger.Panic(err)
		}

		server := &http.Server{
			Addr:         fmt.Sprintf(":%d", options.Port),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  30 * time.Second,
			ErrorLog:     a.Logger,
			Handler:      router,
		}

		// Tell the CLI how to start your router.
		hooks.OnStart(func() {
			logger.Print("API started...")

			server.ListenAndServe()
		})
	})

	cli.Run()
}
