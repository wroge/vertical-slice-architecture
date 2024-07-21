package app

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/wroge/sqlt"
)

type (
	Book struct {
		ID            uuid.UUID `json:"id" required:"true"`
		Title         string    `json:"title" doc:"Title" example:"Titel" required:"true"`
		NumberOfPages int64     `json:"number_of_pages" required:"true"`
		Authors       Authors   `json:"authors,omitempty" required:"true"`
		PublishedAt   time.Time `json:"published_at" required:"true"`
	}

	GetBooksOutputBody struct {
		Total int64  `json:"total" required:"true"`
		Books []Book `json:"books" required:"true"`
	}

	Author struct {
		ID   uuid.UUID `json:"id" required:"true"`
		Name string    `json:"name" required:"true"`
	}

	Authors []Author
)

func (a *Authors) UnmarshalJSON(data []byte) error {
	var into []Author

	if err := json.Unmarshal(data, &into); err != nil {
		return err
	}

	*a = into

	return nil
}

func (a *App) GetBooks(api huma.API) {
	type (
		Input struct {
			Sort      string `query:"sort" doc:"Sort column" default:"id" enum:"id,title,number_of_pages,published_at"`
			Direction string `query:"direction" doc:"direction" enum:"asc,desc"`
			Search    string `query:"search" doc:"Search term"`
			Limit     uint32 `query:"limit" doc:"Limit"`
			Offset    uint32 `query:"offset" doc:"Offset"`
		}

		Output struct {
			Body GetBooksOutputBody `contentType:"application/json" required:"true"`
		}
	)

	a.Template.New("search_filter").MustParse(` 
		{{ if eq Dialect "postgres" }}
			POSITION({{ . }} IN books.title) > 0
		{{ else }}
			instr(books.title, {{ . }}) > 0
		{{ end }}
		OR
		books.id IN (
			SELECT book_authors.book_id
			FROM book_authors
			JOIN authors ON authors.id = book_authors.author_id
			WHERE 
			{{ if eq Dialect "postgres" }}
				POSITION({{ . }} IN authors.name) > 0
			{{ else }}
				instr(authors.name, {{ . }}) > 0
			{{ end }}
		)
	`)

	queryTotal := a.Template.New("query_total").MustParse(`
		SELECT COUNT(*)
		FROM books
		LEFT JOIN book_authors ON book_authors.book_id = books.id
		LEFT JOIN authors ON authors.id = book_authors.author_id
		{{ if .Search }}
			WHERE {{ template "search_filter" .Search }}
		{{ end }};
	`)

	query := a.Template.New("query").MustParse(`
		SELECT
			{{ sqlt.Scanner Dest.ID "books.id" }},
			{{ sqlt.String Dest.Title "books.title" }},
			{{ sqlt.Int64 Dest.NumberOfPages "books.number_of_pages" }},
			{{ sqlt.Time Dest.PublishedAt "books.published_at" }},
			{{ if eq Dialect "postgres" }}
				{{ sqlt.JSON Dest.Authors "json_agg(json_build_object('id', authors.id, 'name', authors.name))" }}
			{{ else }}
				{{ sqlt.JSON Dest.Authors "json_group_array(json_object('id', authors.id, 'name', authors.name))" }}
			{{ end }}
		FROM books
		LEFT JOIN book_authors ON book_authors.book_id = books.id
		LEFT JOIN authors ON authors.id = book_authors.author_id
		{{ if .Search }}
			WHERE {{ template "search_filter" .Search }}
		{{ end }}
		GROUP BY books.id, books.title, books.number_of_pages, books.published_at
		{{ if .Sort }}
			ORDER BY
			{{ if eq .Sort "id" }}
				books.id
				{{ else if eq .Sort "title" }}
				books.title
				{{ else if eq .Sort "number_of_pages" }}
				books.number_of_pages
				{{ else if eq .Sort "published_at" }}
				books.published_at
			{{ end }}
			{{ if eq .Direction "desc" }}
				DESC NULLS LAST
				{{ else }}
				ASC NULLS LAST
			{{ end }}
		{{ end }}
		{{ if .Limit }}
			LIMIT {{ .Limit }}
		{{ end }}
		{{ if .Offset }}
			OFFSET {{ .Offset }}
		{{ end }};
	`)

	op := huma.Operation{
		Method:          http.MethodGet,
		Path:            "/books",
		DefaultStatus:   http.StatusOK,
		MaxBodyBytes:    1 << 20, // 1MB
		BodyReadTimeout: time.Second / 2,
		Errors:          []int{http.StatusInternalServerError},
		Summary:         "Query Books",
		Description:     "Query Books",
	}

	huma.Register(api, op, func(ctx context.Context, input *Input) (*Output, error) {
		total, err := sqlt.FetchFirst[int64](ctx, a.DB, queryTotal, input)
		if err != nil {
			a.Logger.Print(err)

			return nil, huma.Error500InternalServerError("internal error")
		}

		books, err := sqlt.FetchAll[Book](ctx, a.DB, query, input)
		if err != nil {
			a.Logger.Print(err)

			return nil, huma.Error500InternalServerError("internal error")
		}

		return &Output{
			Body: GetBooksOutputBody{
				Total: total,
				Books: books,
			},
		}, nil
	})
}
