package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/wroge/sqlt"
	"github.com/wroge/vertical-slice-architecture/app"
)

func main() {
	db, err := sql.Open("pgx", "host=localhost port=5432 user=user password=password dbname=db sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Cannot connect to database", err)
	}
	fmt.Println("Successfully connected to the database!")

	app := app.App{
		Template: sqlt.New("db", "$", true).Value("Dialect", "postgres").HandleErr(func(err error, runner *sqlt.Runner) error {
			if errors.Is(err, sql.ErrNoRows) {
				// ignore ErrNoRows
				return nil
			}

			// Put logging logic here
			fmt.Println(runner.SQL.String(), runner.Args)

			return err
		}),
		DB:     db,
		Logger: log.New(os.Stdout, "book api - ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	app.Run()
}
