package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-sqlt/sqlt"
	"github.com/google/uuid"
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

		BookAuthors struct {
			BookID    uuid.UUID
			AuthorIDs []uuid.UUID
		}
	)

	insertAuthors := sqlt.All[[]string, uuid.UUID](a.Config, sqlt.Parse(`
		INSERT INTO authors (id, name) VALUES
			{{ range $i, $a := . }} 
				{{ if $i }}, {{ end }}
				({{ uuidv4 }}, {{ $a }})
			{{ end }}
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id;
	`))

	insertBook := sqlt.One[PostBooksInputBody, uuid.UUID](a.Config, sqlt.Parse(`
		INSERT INTO books (id, title, published_at, number_of_pages) VALUES
			({{ uuidv4 }}, {{ .Title }}, {{ .PublishedAt }}, {{ .NumberOfPages }})
		RETURNING id;
	`))

	insertBookAuthors := sqlt.Exec[BookAuthors](a.Config, sqlt.Parse(`
		INSERT INTO book_authors (book_id, author_id) VALUES
		{{ range $i, $a := .AuthorIDs }} 
			{{ if $i }}, {{ end }}
			({{ $.BookID }}, {{ $a }})
		{{ end }};
	`))

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

	huma.Register(api, op, func(ctx context.Context, input *Input) (output *Output, err error) {
		tx, err := a.DB.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}

		defer func() {
			if err != nil {
				err = errors.Join(err, tx.Rollback())
			} else {
				err = tx.Commit()
			}

			if err != nil {
				a.Logger.Info("insert book", "err", err.Error())

				err = huma.Error500InternalServerError("server error")
			}
		}()

		id, err := insertBook.Exec(ctx, tx, input.Body)
		if err != nil {
			return nil, err
		}

		if len(input.Body.Authors) > 0 {
			authors, err := insertAuthors.Exec(ctx, tx, input.Body.Authors)
			if err != nil {
				return nil, err
			}

			_, err = insertBookAuthors.Exec(ctx, tx, BookAuthors{
				BookID:    id,
				AuthorIDs: authors,
			})
			if err != nil {
				return nil, err
			}
		}

		return &Output{
			Body: PostBooksOutputBody{
				ID: id,
			},
		}, nil
	})
}
