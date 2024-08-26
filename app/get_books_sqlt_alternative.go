package app

import (
	"context"
	"net/http"
	"text/template"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/wroge/sqlt"
)

func (a *App) GetBooksSqltAlternative(api huma.API) {
	a.Template.Funcs(template.FuncMap{
		"ScanBooks": sqlt.ScanJSON[[]Book],
	})

	query := a.Template.New("query").MustParse(`
        WITH filtered_books AS (
            SELECT books.id, books.title, books.number_of_pages
                {{ if Postgres }}
                    , to_char(books.published_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS published_at
                    , json_agg(json_build_object('id', authors.id, 'name', authors.name)) AS authors
                {{ else }}
                    , strftime('%Y-%m-%dT%H:%M:%SZ', books.published_at) AS published_at
                    , json_group_array(json_object('id', authors.id, 'name', authors.name)) AS authors
                {{ end }} FROM books
            LEFT JOIN book_authors ON book_authors.book_id = books.id
            LEFT JOIN authors ON authors.id = book_authors.author_id
            {{ if .Search }} WHERE 
                    ({{ if Postgres }} books.title ILIKE '%' || {{ .Search }} || '%'
                    {{ else }} books.title LIKE '%' || {{ .Search }} || '%' {{ end }})
                OR EXISTS (
                    SELECT 1 FROM book_authors JOIN authors ON authors.id = book_authors.author_id
                    WHERE book_authors.book_id = books.id
                    AND ({{ if Postgres }} authors.name ILIKE '%' || {{ .Search }} || '%'
                        {{ else }} authors.name LIKE '%' || {{ .Search }} || '%' {{ end }})
                )
            {{ end }} GROUP BY books.id, books.title, books.number_of_pages, books.published_at
        ),
        paginated_books AS (
            SELECT id, title, number_of_pages, published_at, authors FROM filtered_books
            {{ if .Sort }} ORDER BY {{ Raw .Sort }} {{ Raw .Direction }} NULLS LAST {{ end }}
            {{ if .Limit }} LIMIT {{ .Limit }} {{ end }}
            {{ if .Offset }} OFFSET {{ .Offset }} {{ end }}
        )
        SELECT
            {{ ScanInt64 Dest.Total "(SELECT COUNT(*) FROM filtered_books)" }}
            {{ if Postgres }}
                {{ ScanBooks Dest.Books ", json_agg(json_build_object('id', paginated_books.id, 'title', paginated_books.title, 'number_of_pages', paginated_books.number_of_pages, 'published_at', paginated_books.published_at, 'authors', paginated_books.authors))" }} 
            {{ else }}
                {{ ScanBooks Dest.Books ", json_group_array(json_object('id', paginated_books.id, 'title', paginated_books.title, 'number_of_pages', paginated_books.number_of_pages, 'published_at', paginated_books.published_at, 'authors', json(paginated_books.authors)))" }} 
            {{ end }}
        FROM paginated_books;
    `)

	op := huma.Operation{
		Method:          http.MethodGet,
		Path:            "/sqlt_alternative/books",
		DefaultStatus:   http.StatusOK,
		MaxBodyBytes:    1 << 20, // 1MB
		BodyReadTimeout: time.Second / 2,
		Errors:          []int{http.StatusInternalServerError},
		Summary:         "Query Books Sqlt Alternative",
		Description:     "Query Books Sqlt Alternative",
	}

	huma.Register(api, op, func(ctx context.Context, input *GetBooksInput) (*GetBooksOutput, error) {
		body, err := sqlt.FetchFirst[GetBooksOutputBody](ctx, query, a.DB, input)
		if err != nil {
			return nil, huma.Error500InternalServerError("internal error")
		}

		return &GetBooksOutput{
			Body: body,
		}, nil
	})
}
