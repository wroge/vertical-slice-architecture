package main

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"math/rand/v2"

	"github.com/brianvoe/gofakeit"
	"github.com/google/uuid"
)

func main() {
	// Seed the random number generator
	gofakeit.Seed(0)
	var (
		buffer  bytes.Buffer
		books   [1000]uuid.UUID
		authors [100]uuid.UUID
	)

	// Generate 1000 books
	for i := range 1000 {
		books[i] = uuid.New()
		title := gofakeit.Sentence(3)         // Generates a fake book title
		numberOfPages := rand.IntN(900) + 100 // Random number between 100 and 999
		publishedAt := randomDate().Format(time.DateOnly)

		buffer.WriteString(fmt.Sprintf("INSERT INTO books (id, title, number_of_pages, published_at) VALUES ('%s', '%s', %d, '%s') ON CONFLICT (id) DO NOTHING;\n",
			books[i], title, numberOfPages, publishedAt))
	}

	// Generate 100 authors
	for i := range 100 {
		authors[i] = uuid.New()
		name := gofakeit.Name() // Generates a fake author name

		buffer.WriteString(fmt.Sprintf("INSERT INTO authors (id, name) VALUES ('%s', '%s') ON CONFLICT (id) DO NOTHING;\n", authors[i], name))
	}

	// Generate book_authors relationships
	for i := range 1000 {
		buffer.WriteString(fmt.Sprintf("INSERT INTO book_authors (book_id, author_id) VALUES ('%s', '%s') ON CONFLICT (book_id, author_id) DO NOTHING;\n", books[i], authors[rand.IntN(100)]))

		for rand.IntN(10) > 5 {
			buffer.WriteString(fmt.Sprintf("INSERT INTO book_authors (book_id, author_id) VALUES ('%s', '%s') ON CONFLICT (book_id, author_id) DO NOTHING;\n", books[i], authors[rand.IntN(100)]))
		}
	}

	if err := os.WriteFile("app/data.sql", buffer.Bytes(), 0777); err != nil {
		panic(err)
	}
}

// randomDate generates a random date between 1950 and 2022
func randomDate() time.Time {
	min := time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC).Unix()
	sec := rand.Int64N(max-min) + min
	return time.Unix(sec, 0)
}
