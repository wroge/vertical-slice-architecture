package app

import (
	"context"
	"encoding/json"
	"net/http"
	"text/template"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/wroge/sqlt"
)

func (a *App) GetBooksSqlt(api huma.API) {
	a.Template.Funcs(template.FuncMap{
		"ScanAuthors": func(dest *[]Author, str string) sqlt.Scanner {
			var data []byte

			return sqlt.Scanner{
				SQL:  str,
				Dest: &data,
				Map: func() error {
					var authors []Author

					if err := json.Unmarshal(data, &authors); err != nil {
						return err
					}

					*dest = authors

					return nil
				},
			}
		},
	})

	a.Template.New("search_filter").MustParse(` 
		{{ if eq Dialect "postgres" }}POSITION({{ . }} IN books.title) > 0
		{{ else }}INSTR(books.title, {{ . }}) > 0
		{{ end }}
		OR books.id IN (
			SELECT book_authors.book_id
			FROM book_authors
			JOIN authors ON authors.id = book_authors.author_id
			WHERE 
			{{ if eq Dialect "postgres" }}POSITION({{ . }} IN authors.name) > 0
			{{ else }}INSTR(authors.name, {{ . }}) > 0
			{{ end }}
		)
	`)

	queryTotal := a.Template.New("query_total").MustParse(`
		SELECT COUNT(DISTINCT books.id) FROM books
		LEFT JOIN book_authors ON book_authors.book_id = books.id
		LEFT JOIN authors ON authors.id = book_authors.author_id
		{{ if .Search }}
			WHERE {{ template "search_filter" .Search }}
		{{ end }};
	`)

	query := a.Template.New("query").MustParse(`
		SELECT
			{{ Scan Dest.ID "books.id" }}
			{{ ScanString Dest.Title ", books.title" }}
			{{ ScanInt64 Dest.NumberOfPages ", books.number_of_pages" }}
			{{ ScanTime Dest.PublishedAt ", books.published_at" }}
			{{ if eq Dialect "postgres" }}
				{{ ScanAuthors Dest.Authors ", json_agg(json_build_object('id', authors.id, 'name', authors.name))" }}
			{{ else }}
				{{ ScanAuthors Dest.Authors ", json_group_array(json_object('id', authors.id, 'name', authors.name))" }}
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
			{{ if eq .Sort "id" }}books.id
			{{ else if eq .Sort "title" }}books.title
			{{ else if eq .Sort "number_of_pages" }}books.number_of_pages
			{{ else if eq .Sort "published_at" }}books.published_at
			{{ else }} {{ fail "invalid sort column" }}
			{{ end }}
			{{ if eq .Direction "desc" }}DESC NULLS LAST
			{{ else }}ASC NULLS LAST
			{{ end }}
		{{ end }}
		{{ if .Limit }}LIMIT {{ .Limit }}{{ end }}
		{{ if .Offset }}OFFSET {{ .Offset }}{{ end }};
	`)

	op := huma.Operation{
		Method:          http.MethodGet,
		Path:            "/sqlt/books",
		DefaultStatus:   http.StatusOK,
		MaxBodyBytes:    1 << 20, // 1MB
		BodyReadTimeout: time.Second / 2,
		Errors:          []int{http.StatusInternalServerError},
		Summary:         "Query Books Sqlt",
		Description:     "Query Books Sqlt",
	}

	huma.Register(api, op, func(ctx context.Context, input *GetBooksInput) (*GetBooksOutput, error) {
		total, err := sqlt.FetchFirst[int64](ctx, queryTotal, a.DB, input)
		if err != nil {
			a.Logger.Print(err)

			return nil, huma.Error500InternalServerError("internal error")
		}

		books, err := sqlt.FetchAll[Book](ctx, query, a.DB, input)
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
