package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type (
	Book struct {
		ID            uuid.UUID `json:"id"`
		Title         string    `json:"title" doc:"Title" example:"Titel"`
		NumberOfPages int64     `json:"number_of_pages"`
		Authors       []Author  `json:"authors,omitempty"`
		PublishedAt   time.Time `json:"published_at"`
	}

	Author struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}

	GetBooksOutputBody struct {
		Total int64  `json:"total"`
		Books []Book `json:"books"`
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

func (a *App) GetBooksStandard(api huma.API) {
	filter := func(search string) (string, []any) {
		if a.Dialect == "postgres" {
			return `(
					POSITION($1 IN books.title) > 0 OR
					books.id IN (
						SELECT book_authors.book_id
						FROM book_authors
						JOIN authors ON authors.id = book_authors.author_id
						WHERE POSITION($2 IN authors.name) > 0
					)
				)`, []any{search, search}
		} else {
			return `(
					INSTR(books.title, ?) > 0 OR
					books.id IN (
						SELECT book_authors.book_id
						FROM book_authors
						JOIN authors ON authors.id = book_authors.author_id
						WHERE INSTR(authors.name, ?) > 0
					)
				)`, []any{search, search}
		}
	}

	queryTotal := func(ctx context.Context, input *GetBooksInput) (int64, error) {
		var (
			sb    strings.Builder
			args  []any
			total int64
		)

		sb.WriteString("SELECT COUNT(DISTINCT books.id) FROM books ")
		sb.WriteString("LEFT JOIN book_authors ON book_authors.book_id = books.id ")
		sb.WriteString("LEFT JOIN authors ON authors.id = book_authors.author_id ")

		if input.Search != "" {
			query, fArgs := filter(input.Search)
			sb.WriteString("WHERE " + query)
			args = append(args, fArgs...)
		}

		query := sb.String()

		err := a.DB.QueryRowContext(ctx, query, args...).Scan(&total)
		if err != nil {
			return 0, err
		}

		return total, nil
	}

	query := func(ctx context.Context, input *GetBooksInput) ([]Book, error) {
		var (
			sb    strings.Builder
			args  []any
			books []Book
		)

		sb.WriteString("SELECT books.id, books.title, books.number_of_pages, books.published_at, ")

		if a.Dialect == "postgres" {
			sb.WriteString("json_agg(json_build_object('id', authors.id, 'name', authors.name)) ")
		} else {
			sb.WriteString("json_group_array(json_object('id', authors.id, 'name', authors.name)) ")
		}
		sb.WriteString(`FROM books
			LEFT JOIN book_authors ON book_authors.book_id = books.id
			LEFT JOIN authors ON authors.id = book_authors.author_id
		`)

		if input.Search != "" {
			query, fArgs := filter(input.Search)
			sb.WriteString("WHERE " + query)
			args = append(args, fArgs...)
		}

		sb.WriteString("GROUP BY books.id, books.title, books.number_of_pages, books.published_at ")

		if input.Sort != "" {
			var direction string
			if input.Direction == "desc" {
				direction = "DESC NULLS LAST"
			} else {
				direction = "ASC NULLS LAST"
			}

			switch input.Sort {
			case "id":
				sb.WriteString("ORDER BY books.id " + direction + " ")
			case "title":
				sb.WriteString("ORDER BY books.title " + direction + " ")
			case "number_of_pages":
				sb.WriteString("ORDER BY books.number_of_pages " + direction + " ")
			case "published_at":
				sb.WriteString("ORDER BY books.published_at " + direction + " ")
			}
		}

		if input.Limit > 0 {
			sb.WriteString("LIMIT ? ")
			args = append(args, input.Limit)
		}

		if input.Offset > 0 {
			sb.WriteString("OFFSET ? ")
			args = append(args, input.Offset)
		}

		query := sb.String()

		rows, err := a.DB.QueryContext(ctx, query, args...)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}

			return nil, err
		}

		defer rows.Close()

		for rows.Next() {
			var (
				authors []byte
				book    Book
			)

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
