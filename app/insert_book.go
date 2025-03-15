package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/wroge/sqlt"
)

func (a *App) InsertBook(api huma.API) {
	type (
		PostBooksInputBody struct {
			PublishedAt   string   `json:"published_at" format:"date" required:"true"`
			Title         string   `json:"title" required:"true"`
			Authors       []string `json:"authors" required:"false"`
			NumberOfPages int64    `json:"number_of_pages" required:"false"`
		}

		PostBooksOutputBody struct {
			ID uuid.UUID `json:"id" required:"true"`
		}

		Input struct {
			Body PostBooksInputBody
		}

		Output struct {
			Body PostBooksOutputBody
		}
	)

	insert := sqlt.Transaction(
		nil,
		sqlt.Exec[PostBooksInputBody](a.Config, sqlt.Parse(`
				{{ if .Authors }}
					INSERT INTO authors (id, name) VALUES
						{{ range $i, $a := .Authors }} 
							{{ if $i }}, {{ end }}
							({{ uuidv4 }}, {{ $a }})
						{{ end }}
					ON CONFLICT (name) DO NOTHING;
				{{ end }}
			`),
		),
		sqlt.All[PostBooksInputBody, uuid.UUID](a.Config, sqlt.Name("AuthorIDs"), sqlt.Parse(`
				{{ if .Authors }}
					SELECT id FROM authors WHERE name IN(
						{{ range $i, $a := .Authors }} 
							{{ if $i }}, {{ end }}
							{{ $a }}
						{{ end }}
					);
				{{ end }}
			`),
		),
		sqlt.One[PostBooksInputBody, uuid.UUID](a.Config, sqlt.Name("BookID"), sqlt.Parse(`
				INSERT INTO books (id, title, published_at, number_of_pages) VALUES
					({{ uuidv4 }}, {{ .Title }}, {{ .PublishedAt }}, {{ .NumberOfPages }})
				RETURNING id;
			`),
		),
		sqlt.Exec[PostBooksInputBody](a.Config, sqlt.Parse(`
				{{ if .Authors }}
					INSERT INTO book_authors (book_id, author_id) VALUES
					{{ range $i, $a := (Context "AuthorIDs") }} 
						{{ if $i }}, {{ end }}
						({{ Context "BookID" }}, {{ $a }})
					{{ end }};
				{{ end }}
			`),
		),
	)

	op := huma.Operation{
		Method:          http.MethodPost,
		Path:            "/books",
		DefaultStatus:   http.StatusCreated,
		MaxBodyBytes:    1 << 20, // 1MB
		BodyReadTimeout: time.Second / 2,
		Errors:          []int{http.StatusInternalServerError},
		Summary:         "Insert Book",
		Description:     "Insert Book",
	}

	huma.Register(api, op, func(ctx context.Context, input *Input) (*Output, error) {
		ctx, err := insert.Exec(ctx, a.DB, input.Body)
		if err != nil {
			fmt.Println(err)

			return nil, huma.Error500InternalServerError("internal error")
		}

		return &Output{
			Body: PostBooksOutputBody{
				ID: ctx.Value(sqlt.ContextKey("BookID")).(uuid.UUID),
			},
		}, nil
	})
}
