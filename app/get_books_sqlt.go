package app

import (
	"context"
	"net/http"
	"text/template"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/wroge/sqlt"
)

func (a *App) GetBooksSqlt(api huma.API) {
	a.Template.Funcs(template.FuncMap{
		"ScanAuthors": sqlt.ScanJSON[[]Author],
	})

	a.Template.New("search_filter").MustParse(` 
		{{ if Postgres }}books.title ILIKE '%' || {{ .Search }} || '%' OR
			EXISTS (
				SELECT 1 FROM book_authors JOIN authors ON authors.id = book_authors.author_id
				WHERE book_authors.book_id = books.id
				AND authors.name ILIKE '%' || {{ .Search }} || '%'
			)
		{{ else }}books.title LIKE '%' || {{ .Search }} || '%' OR
			EXISTS (
				SELECT 1 FROM book_authors JOIN authors ON authors.id = book_authors.author_id
				WHERE book_authors.book_id = books.id
				AND authors.name LIKE '%' || {{ .Search }} || '%'
			)
		{{ end }}
	`)

	queryTotal := a.Template.New("query_total").MustParse(`
		SELECT COUNT(DISTINCT books.id) FROM books
		LEFT JOIN book_authors ON book_authors.book_id = books.id
		LEFT JOIN authors ON authors.id = book_authors.author_id
		{{ if .Search }} WHERE {{ template "search_filter" . }}{{ end }};
	`)

	query := a.Template.New("query").MustParse(`
		SELECT
			{{ Scan Dest.ID "books.id" }}
			{{ ScanString Dest.Title ", books.title" }}
			{{ ScanInt64 Dest.NumberOfPages ", books.number_of_pages" }}
			{{ ScanTime Dest.PublishedAt ", books.published_at" }}
			{{ if Postgres }}
				{{ ScanAuthors Dest.Authors ", json_agg(json_build_object('id', authors.id, 'name', authors.name))" }}
			{{ else }}
				{{ ScanAuthors Dest.Authors ", json_group_array(json_object('id', authors.id, 'name', authors.name))" }}
			{{ end }}
		FROM books
		LEFT JOIN book_authors ON book_authors.book_id = books.id
		LEFT JOIN authors ON authors.id = book_authors.author_id
		{{ if .Search }} WHERE {{ template "search_filter" . }}
		{{ end }} GROUP BY books.id, books.title, books.number_of_pages, books.published_at
		{{ if .Sort }} ORDER BY books.{{ Raw .Sort }} {{ Raw .Direction }} NULLS LAST {{ end }}
		{{ if .Limit }} LIMIT {{ .Limit }}{{ end }}
		{{ if .Offset }} OFFSET {{ .Offset }}{{ end }};
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
		total, err := sqlt.FetchOne[int64](ctx, queryTotal, a.DB, input)
		if err != nil {
			return nil, huma.Error500InternalServerError("internal error")
		}

		books, err := sqlt.FetchAll[Book](ctx, query, a.DB, input)
		if err != nil {
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
