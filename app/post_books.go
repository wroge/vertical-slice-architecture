package app

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/wroge/sqlt"
)

func (a *App) PostBooks(api huma.API) {
	type (
		PostBooksInputBody struct {
			Title         string
			NumberOfPages int64
			Authors       []string
			PublishedAt   time.Time
		}

		PostBooksOutputBody struct {
			ID uuid.UUID
		}

		Input struct {
			Body PostBooksInputBody
		}

		Output struct {
			Body PostBooksOutputBody
		}
	)

	insertAuthors := a.Template.New("insert_authors").MustParse(`
		INSERT INTO authors (id, name) VALUES
		{{ range $i, $a := . }} {{ if $i }}, {{ end }}
			({{ uuidv4 }}, {{ $a }})
		{{ end }}
		ON CONFLICT (name) DO NOTHING;;
	`)

	queryAuthors := a.Template.New("query_authors").MustParse(`
		SELECT id FROM authors WHERE name IN(
		{{ range $i, $a := . }} {{ if $i }}, {{ end }}
			{{ $a }}
		{{ end }});
	`)

	insertBook := a.Template.New("insert_book").MustParse(`
		INSERT INTO books (id, title, number_of_pages, published_at) VALUES
			({{ .ID }},{{ .Title }},{{ .NumberOfPages }},{{ .PublishedAt }})
	`)

	insertBookAuthors := a.Template.New("insert_book_authors").MustParse(`
		INSERT INTO book_authors (book_id, author_id) VALUES
		{{ range $i, $a := .AuthorIDs }} {{ if $i }}, {{ end }}
			({{ $.BookID }}, {{ $a }})
		{{ end }};
	`)

	op := huma.Operation{
		Method:          http.MethodPost,
		Path:            "/books",
		DefaultStatus:   http.StatusCreated,
		MaxBodyBytes:    1 << 20, // 1MB
		BodyReadTimeout: time.Second / 2,
		Errors:          []int{http.StatusBadRequest, http.StatusInternalServerError},
		Summary:         "Insert Book",
		Description:     "Insert Book",
	}

	huma.Register(api, op, func(ctx context.Context, input *Input) (*Output, error) {
		if input.Body.Title == "" {
			return nil, huma.Error400BadRequest("please provide a title")
		}

		if len(input.Body.Authors) == 0 {
			return nil, huma.Error400BadRequest("please provide an author")
		}

		bookID := uuid.New()

		err := sqlt.InTx(ctx, nil, a.DB, func(db sqlt.DB) error {
			_, err := sqlt.Exec(ctx, db, insertBook, map[string]any{
				"ID":            bookID,
				"Title":         input.Body.Title,
				"NumberOfPages": input.Body.NumberOfPages,
				"PublishedAt":   input.Body.PublishedAt,
			})
			if err != nil {
				return err
			}

			_, err = sqlt.Exec(ctx, db, insertAuthors, input.Body.Authors)
			if err != nil {
				return err
			}

			authorIDs, err := sqlt.QueryAll[uuid.UUID](ctx, db, queryAuthors, input.Body.Authors)
			if err != nil {
				return err
			}

			_, err = sqlt.Exec(ctx, db, insertBookAuthors, map[string]any{
				"AuthorIDs": authorIDs,
				"BookID":    bookID,
			})
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			a.Logger.Panic(err)

			return nil, huma.Error500InternalServerError("internal error")
		}

		return &Output{
			Body: PostBooksOutputBody{
				ID: bookID,
			},
		}, nil
	})
}
