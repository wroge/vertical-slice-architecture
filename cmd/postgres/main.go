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
		Template: sqlt.New("db").Dollar().Value("Dialect", "postgres").HandleErr(func(err sqlt.Error) error {
			if errors.Is(err.Err, sql.ErrNoRows) {
				// ignore ErrNoRows
				return nil
			}

			// Put logging logic here
			fmt.Println(err.SQL, err.Args)

			return err.Err
		}),
		DB:     db,
		Logger: log.New(os.Stdout, "book api - ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	app.Run()
}
