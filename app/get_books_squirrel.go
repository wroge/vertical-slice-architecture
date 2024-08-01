package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type (
	Book struct {
		ID            uuid.UUID `json:"id" required:"true"`
		Title         string    `json:"title" doc:"Title" example:"Titel" required:"true"`
		NumberOfPages int64     `json:"number_of_pages" required:"true"`
		Authors       []Author  `json:"authors,omitempty" required:"true"`
		PublishedAt   time.Time `json:"published_at" required:"true"`
	}

	Author struct {
		ID   uuid.UUID `json:"id" required:"true"`
		Name string    `json:"name" required:"true"`
	}

	GetBooksOutputBody struct {
		Total int64  `json:"total" required:"true"`
		Books []Book `json:"books" required:"true"`
	}

	GetBooksInput struct {
		Sort      string `query:"sort" doc:"Sort column" default:"id" enum:"id,title,number_of_pages,published_at"`
		Direction string `query:"direction" doc:"direction" enum:"asc,desc"`
		Search    string `query:"search" doc:"Search term"`
		Limit     int    `query:"limit" doc:"Limit" required:"true" minimum:"1" maximum:"100" default:"10"`
		Offset    int    `query:"offset" doc:"Offset" minimum:"0"`
	}

	GetBooksOutput struct {
		Body GetBooksOutputBody `contentType:"application/json" required:"true"`
	}
)

func (a *App) GetBooksSquirrel(api huma.API) {
	filter := func(search string) squirrel.Sqlizer {
		if a.Dialect == "postgres" {
			return squirrel.Or{
				squirrel.Expr("POSITION(? IN books.title) > 0", search),
				squirrel.Expr(`
					books.id IN (
						SELECT book_authors.book_id
						FROM book_authors
						JOIN authors ON authors.id = book_authors.author_id
						WHERE POSITION(? IN authors.name) > 0
					)
				`, search),
			}
		}

		return squirrel.Or{
			squirrel.Expr("INSTR(books.title, ?) > 0", search),
			squirrel.Expr(`
				books.id IN (
					SELECT book_authors.book_id
					FROM book_authors
					JOIN authors ON authors.id = book_authors.author_id
					WHERE INSTR(authors.name, ?) > 0
				)
			`, search),
		}
	}

	queryTotal := func(ctx context.Context, input *GetBooksInput) (int64, error) {
		sb := squirrel.Select("COUNT(DISTINCT books.id)").From("books").
			LeftJoin("book_authors ON book_authors.book_id = books.id").
			LeftJoin("authors ON authors.id = book_authors.author_id")

		if input.Search != "" {
			sb = sb.Where(filter(input.Search))
		}

		if a.Dialect == "postgres" {
			sb = sb.PlaceholderFormat(squirrel.Dollar)
		}

		var total int64

		if err := sb.RunWith(a.DB).QueryRowContext(ctx).Scan(&total); err != nil {
			return 0, err
		}

		return total, nil
	}

	query := func(ctx context.Context, input *GetBooksInput) ([]Book, error) {
		sb := squirrel.Select(
			"books.id",
			"books.title",
			"books.number_of_pages",
			"books.published_at",
		)

		if a.Dialect == "postgres" {
			sb = sb.Column("json_agg(json_build_object('id', authors.id, 'name', authors.name))")
		} else {
			sb = sb.Column("json_group_array(json_object('id', authors.id, 'name', authors.name))")
		}

		sb = sb.From("books").
			LeftJoin("book_authors ON book_authors.book_id = books.id").
			LeftJoin("authors ON authors.id = book_authors.author_id")

		if input.Search != "" {
			sb = sb.Where(filter(input.Search))
		}

		sb = sb.GroupBy("books.id", "books.title", "books.number_of_pages", "books.published_at")

		if input.Sort != "" {
			var direction string

			if input.Direction == "desc" {
				direction = "DESC NULLS LAST"
			} else {
				direction = "ASC NULLS LAST"
			}

			switch input.Sort {
			case "id":
				sb = sb.OrderBy("books.id " + direction)
			case "title":
				sb = sb.OrderBy("books.title " + direction)
			case "number_of_pages":
				sb = sb.OrderBy("books.number_of_pages " + direction)
			case "published_at":
				sb = sb.OrderBy("books.published_at " + direction)
			}
		}

		if input.Limit > 0 {
			sb = sb.Limit(uint64(input.Limit))
		}

		if input.Offset > 0 {
			sb = sb.Offset(uint64(input.Offset))
		}

		if a.Dialect == "postgres" {
			sb = sb.PlaceholderFormat(squirrel.Dollar)
		}

		rows, err := sb.RunWith(a.DB).QueryContext(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}

			return nil, err
		}

		defer rows.Close()

		var (
			authors []byte
			book    Book
			books   []Book
		)

		for rows.Next() {
			if err := rows.Scan(&book.ID, &book.Title, &book.NumberOfPages, &book.PublishedAt, &authors); err != nil {
				return nil, err
			}

			if err := json.Unmarshal(authors, &book.Authors); err != nil {
				return nil, err
			}

			books = append(books, book)
		}

		return books, nil
	}

	op := huma.Operation{
		Method:          http.MethodGet,
		Path:            "/squirrel/books",
		DefaultStatus:   http.StatusOK,
		MaxBodyBytes:    1 << 20, // 1MB
		BodyReadTimeout: time.Second / 2,
		Errors:          []int{http.StatusInternalServerError},
		Summary:         "Query Books Standard",
		Description:     "Query Books Standard",
	}

	huma.Register(api, op, func(ctx context.Context, input *GetBooksInput) (*GetBooksOutput, error) {
		total, err := queryTotal(ctx, input)
		if err != nil {
			a.Logger.Print(err)

			return nil, huma.Error500InternalServerError("internal error")
		}

		books, err := query(ctx, input)
		if err != nil {
			a.Logger.Print(err)

			return nil, huma.Error500InternalServerError("internal error")
		}

		return &GetBooksOutput{
			Body: GetBooksOutputBody{
				Total: total,
				Books: books,
			},
		}, nil
	})
}
