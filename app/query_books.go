package app

import (
	"context"
	"net/http"
	"text/template"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/wroge/sqlt"
)

type (
	Book struct {
		PublishedAt   string    `json:"published_at" format:"date"`
		Title         string    `json:"title" doc:"Title"`
		Authors       []Author  `json:"authors,omitempty"`
		NumberOfPages int64     `json:"number_of_pages"`
		ID            uuid.UUID `json:"id"`
	}

	Author struct {
		Name string    `json:"name"`
		ID   uuid.UUID `json:"id"`
	}

	GetBooksOutputBody struct {
		Books []Book `json:"books"`
		Total int64  `json:"total"`
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

func (a *App) GetBooksSqltAlternative(api huma.API) {
	query := sqlt.QueryStmt[*GetBooksInput, GetBooksOutputBody](
		a.Config,
		sqlt.Funcs(template.FuncMap{"ScanBooks": sqlt.ScanJSON[[]Book]}),
		sqlt.Parse(`
			WITH filtered_books AS (
				SELECT books.id, books.title, books.number_of_pages
					{{ if eq Dialect "postgres" }}
						, to_char(books.published_at, 'YYYY-MM-DD') AS published_at
						, jsonb_agg(jsonb_build_object('id', authors.id, 'name', authors.name)) AS authors
					{{ else if eq Dialect "sqlite" }}
						, strftime('%Y-%m-%d', books.published_at) AS published_at
						, json_group_array(json_object('id', authors.id, 'name', authors.name)) AS authors
					{{ else }}
						{{ fail "invalid dialect" }}
					{{ end }} 
				FROM books
				LEFT JOIN book_authors ON book_authors.book_id = books.id
				LEFT JOIN authors ON authors.id = book_authors.author_id
				{{ with (lower .Search) }} 
					WHERE     
					{{ if eq Dialect "postgres" }}
						POSITION({{ . }} IN LOWER(books.title)) > 0
					{{ else if eq Dialect "sqlite" }} 
						INSTR(LOWER(books.title), {{ . }}) 
					{{ else }}
						{{ fail "invalid dialect" }}
					{{ end }}
					OR EXISTS (
						SELECT 1 FROM book_authors JOIN authors ON authors.id = book_authors.author_id
						WHERE book_authors.book_id = books.id
						AND (
							{{ if eq Dialect "postgres" }} 
								POSITION({{ . }} IN LOWER(authors.name)) > 0
							{{ else if eq Dialect "sqlite" }} 
								INSTR(LOWER(authors.name), {{ . }}) 
							{{ else }}
								{{ fail "invalid dialect" }}
							{{ end }}
						)
					)
				{{ end }} 
				GROUP BY books.id, books.title, books.number_of_pages, books.published_at
			),
			paginated_books AS (
				SELECT id, title, number_of_pages, published_at, authors FROM filtered_books
				{{ if .Sort }} 
					ORDER BY {{ Raw .Sort }} {{ Raw .Direction }} NULLS LAST 
				{{ end }}
				{{ if .Limit }} 
					LIMIT {{ .Limit }} 
				{{ end }}
				{{ if .Offset }} 
					OFFSET {{ .Offset }} 
				{{ end }}
			)
			SELECT
				{{ ScanInt64 Dest.Total "(SELECT COUNT(*) FROM filtered_books)" }}
				{{ if eq Dialect "postgres" }}
					{{ ScanBooks Dest.Books ", jsonb_agg(jsonb_build_object('id', paginated_books.id, 'title', paginated_books.title, 'number_of_pages', paginated_books.number_of_pages, 'published_at', paginated_books.published_at, 'authors', paginated_books.authors))" }} 
				{{ else if eq Dialect "sqlite" }}
					{{ ScanBooks Dest.Books ", json_group_array(json_object('id', paginated_books.id, 'title', paginated_books.title, 'number_of_pages', paginated_books.number_of_pages, 'published_at', paginated_books.published_at, 'authors', json(paginated_books.authors)))" }} 
				{{ else }}
					{{ fail "invalid dialect" }}
				{{ end }}
			FROM paginated_books;
		`),
	)

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

	huma.Register(api, op, func(ctx context.Context, input *GetBooksInput) (*GetBooksOutput, error) {
		body, err := query.First(ctx, a.DB, input)
		if err != nil {
			return nil, huma.Error500InternalServerError("internal error")
		}

		return &GetBooksOutput{
			Body: body,
		}, nil
	})
}
