package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/wroge/sqlt"
)

type (
	Book struct {
		PublishedAt   time.Time          `json:"published_at"`
		Title         string             `json:"title" doc:"Title"`
		Authors       sqlt.Slice[Author] `json:"authors,omitempty"`
		NumberOfPages int64              `json:"number_of_pages"`
		ID            uuid.UUID          `json:"id"`
	}

	Author struct {
		Name string    `json:"name"`
		ID   uuid.UUID `json:"id"`
	}

	GetBooksOutputBody struct {
		Books sqlt.Slice[Book] `json:"books"`
		Total int64            `json:"total"`
	}

	GetBooksInput struct {
		Sort      string `query:"sort" doc:"Sort column" default:"id" enum:"id,title,number_of_pages,published_at"`
		Direction string `query:"direction" doc:"direction" enum:"asc,desc"`
		Search    string `query:"search" doc:"Search term"`
		Limit     uint64 `query:"limit" doc:"Limit" required:"true" minimum:"1" maximum:"100" default:"10"`
		Offset    uint64 `query:"offset" doc:"Offset" minimum:"0"`
	}

	GetBooksOutput struct {
		Body GetBooksOutputBody `contentType:"application/json" required:"true"`
	}
)

func (a *App) GetBooksStandard(api huma.API) {
	filter := func(search string) (string, []any) {
		if a.Dialect == Postgres {
			return `books.title ILIKE '%' || ? || '%' OR
			EXISTS (
				SELECT 1
				FROM book_authors
				JOIN authors ON authors.id = book_authors.author_id
				WHERE book_authors.book_id = books.id
				AND authors.name ILIKE '%' || ? || '%'
			)`, []any{search, search}
		} else {
			return `books.title LIKE '%' || ? || '%' OR
			EXISTS (
				SELECT 1
				FROM book_authors
				JOIN authors ON authors.id = book_authors.author_id
				WHERE book_authors.book_id = books.id
				AND authors.name LIKE '%' || ? || '%'
			)`, []any{search, search}
		}
	}

	queryTotal := func(ctx context.Context, input *GetBooksInput) (int64, error) {
		queryBuilder := squirrel.Select("COUNT(DISTINCT books.id)").
			From("books").
			LeftJoin("book_authors ON book_authors.book_id = books.id").
			LeftJoin("authors ON authors.id = book_authors.author_id")

		if input.Search != "" {
			searchQuery, fArgs := filter(input.Search)
			queryBuilder = queryBuilder.Where(searchQuery, fArgs...)
		}

		if a.Dialect == Postgres {
			queryBuilder = queryBuilder.PlaceholderFormat(squirrel.Dollar)
		}

		var total int64
		err := queryBuilder.RunWith(a.DB).ScanContext(ctx, &total)
		if err != nil {
			return 0, err
		}

		return total, nil
	}

	query := func(ctx context.Context, input *GetBooksInput) ([]Book, error) {
		queryBuilder := squirrel.Select("books.id", "books.title", "books.number_of_pages", "books.published_at")

		if a.Dialect == Postgres {
			queryBuilder = queryBuilder.Column("jsonb_agg(jsonb_build_object('id', authors.id, 'name', authors.name))")
		} else {
			queryBuilder = queryBuilder.Column("json_group_array(json_object('id', authors.id, 'name', authors.name))")
		}

		queryBuilder = queryBuilder.From("books").
			LeftJoin("book_authors ON book_authors.book_id = books.id").
			LeftJoin("authors ON authors.id = book_authors.author_id")

		if input.Search != "" {
			searchQuery, fArgs := filter(input.Search)
			queryBuilder = queryBuilder.Where(searchQuery, fArgs...)
		}

		queryBuilder = queryBuilder.GroupBy("books.id", "books.title", "books.number_of_pages", "books.published_at")

		if input.Sort != "" {
			queryBuilder = queryBuilder.OrderBy(fmt.Sprintf("books.%s %s NULLS LAST", input.Sort, input.Direction))
		}

		if input.Limit > 0 {
			queryBuilder = queryBuilder.Limit(input.Limit)
		}

		if input.Offset > 0 {
			queryBuilder = queryBuilder.Offset(input.Offset)
		}

		if a.Dialect == Postgres {
			queryBuilder = queryBuilder.PlaceholderFormat(squirrel.Dollar)
		}

		rows, err := queryBuilder.RunWith(a.DB).QueryContext(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}

			return nil, err
		}

		defer rows.Close()

		var (
			authorsData []byte
			book        Book
			books       []Book
		)

		for rows.Next() {
			if err := rows.Scan(&book.ID, &book.Title, &book.NumberOfPages, &book.PublishedAt, &authorsData); err != nil {
				return nil, err
			}

			var authors []Author

			if err := json.Unmarshal(authorsData, &authors); err != nil {
				return nil, err
			}

			book.Authors = authors

			books = append(books, book)
		}

		return books, nil
	}

	op := huma.Operation{
		Method:          http.MethodGet,
		Path:            "/standard/books",
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
			a.Logger.Error(err.Error())

			return nil, huma.Error500InternalServerError("internal error")
		}

		books, err := query(ctx, input)
		if err != nil {
			a.Logger.Error(err.Error())

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
