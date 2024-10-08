package app

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/wroge/sqlt"
)

func (a *App) GetBooksSqlt(api huma.API) {
	a.Template.New("search_filter").MustParse(` 
		{{ if Postgres }} POSITION({{ . }} IN LOWER(books.title)) > 0
		{{ else }} INSTR(LOWER(books.title), {{ . }}) 
		{{ end }}
		OR EXISTS (
			SELECT 1 FROM book_authors JOIN authors ON authors.id = book_authors.author_id
			WHERE book_authors.book_id = books.id
			AND ({{ if Postgres }} POSITION({{ . }} IN LOWER(authors.name)) > 0
				{{ else }} INSTR(LOWER(authors.name), {{ . }}) {{ end }})
		)
	`)

	queryTotal := sqlt.MustType[int64, *GetBooksInput](a.Template.New("query_total").MustParse(`
		SELECT COUNT(DISTINCT books.id) FROM books
		LEFT JOIN book_authors ON book_authors.book_id = books.id
		LEFT JOIN authors ON authors.id = book_authors.author_id
		{{ with (lower .Search) }} WHERE {{ template "search_filter" . }}{{ end }};
	`))

	query := sqlt.MustType[Book, *GetBooksInput](a.Template.New("query").MustParse(`
		SELECT
			{{ Scan Dest.ID "books.id" }}
			{{ ScanString Dest.Title ", books.title" }}
			{{ ScanInt64 Dest.NumberOfPages ", books.number_of_pages" }}
			{{ ScanString Dest.PublishedAt ", strftime('%Y-%m-%d', books.published_at)" }}
			{{ if Postgres }}
				{{ ScanJSON Dest.Authors ", jsonb_agg(jsonb_build_object('id', authors.id, 'name', authors.name))" }}
			{{ else }}
				{{ ScanJSON Dest.Authors ", json_group_array(json_object('id', authors.id, 'name', authors.name))" }}
			{{ end }}
		FROM books
		LEFT JOIN book_authors ON book_authors.book_id = books.id
		LEFT JOIN authors ON authors.id = book_authors.author_id
		{{ with (lower .Search) }} WHERE {{ template "search_filter" . }}
		{{ end }} GROUP BY books.id, books.title, books.number_of_pages, books.published_at
		{{ if .Sort }} ORDER BY books.{{ Raw .Sort }} {{ Raw .Direction }} NULLS LAST {{ end }}
		{{ if .Limit }} LIMIT {{ .Limit }}{{ end }}
		{{ if .Offset }} OFFSET {{ .Offset }}{{ end }};
	`))

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
		var (
			total int64
			books = make([]Book, 0, input.Limit)
		)

		err := queryTotal.ScanOne(ctx, a.DB, input, &total)
		if err != nil {
			return nil, huma.Error500InternalServerError("internal error")
		}

		err = query.ScanAll(ctx, a.DB, input, &books)
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
