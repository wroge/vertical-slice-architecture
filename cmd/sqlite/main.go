package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wroge/sqlt"
	"github.com/wroge/vertical-slice-architecture/app"
)

func main() {
	db, err := sql.Open("sqlite3", "file:test.db?cache=shared&mode=memory")
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Cannot connect to database", err)
	}
	fmt.Println("Successfully connected to the database!")

	app := app.App{
		Template: sqlt.New("db", "?", false).Value("Dialect", "sqlite"),
		DB:       db,
		Logger:   log.New(os.Stdout, "book api - ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	app.Run()
}
