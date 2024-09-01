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
			PublishedAt   string   `json:"published_at" format:"date" required:"true"`
			Title         string   `json:"title" required:"true"`
			Authors       []string `json:"authors" required:"true"`
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

	insertAuthors := a.Template.New("insert_authors").MustParse(`
		INSERT INTO authors (id, name) VALUES
		{{ range $i, $a := . }} {{ if $i }}, {{ end }}
			({{ uuidv4 }}, {{ $a }})
		{{ end }}
		ON CONFLICT (name) DO NOTHING;;
	`)

	queryAuthors := sqlt.DestParam[uuid.UUID, []string](a.Template.New("query_authors").MustParse(`
		SELECT id FROM authors WHERE name IN(
		{{ range $i, $a := . }} {{ if $i }}, {{ end }}
			{{ $a }}
		{{ end }});
	`))

	insertBook := sqlt.DestParam[uuid.UUID, PostBooksInputBody](a.Template.New("insert_book").MustParse(`
		INSERT INTO books (id, title, published_at, number_of_pages) VALUES
			({{ uuidv4 }},{{ .Title }},{{ .PublishedAt }}, {{ .NumberOfPages }})
		RETURNING id;
	`))

	type InsertBookAuthor struct {
		BookID    uuid.UUID
		AuthorIDs []uuid.UUID
	}

	insertBookAuthors := sqlt.Param[InsertBookAuthor](a.Template.New("insert_book_authors").MustParse(`
		INSERT INTO book_authors (book_id, author_id) VALUES
		{{ range $i, $a := .AuthorIDs }} {{ if $i }}, {{ end }}
			({{ $.BookID }}, {{ $a }})
		{{ end }};
	`))

	op := huma.Operation{
		Method:          http.MethodPost,
		Path:            "/sqlt/books",
		DefaultStatus:   http.StatusCreated,
		MaxBodyBytes:    1 << 20, // 1MB
		BodyReadTimeout: time.Second / 2,
		Errors:          []int{http.StatusInternalServerError},
		Summary:         "Insert Book Sqlt",
		Description:     "Insert Book Sqlt",
	}

	huma.Register(api, op, func(ctx context.Context, input *Input) (*Output, error) {
		var (
			id  uuid.UUID
			err error
		)

		err = sqlt.InTx(ctx, nil, a.DB, func(db sqlt.DB) error {
			id, err = insertBook.One(ctx, db, input.Body)
			if err != nil {
				return err
			}

			_, err = insertAuthors.Exec(ctx, db, input.Body.Authors)
			if err != nil {
				return err
			}

			authorIDs, err := queryAuthors.All(ctx, db, input.Body.Authors)
			if err != nil {
				return err
			}

			_, err = insertBookAuthors.Exec(ctx, db, InsertBookAuthor{
				AuthorIDs: authorIDs,
				BookID:    id,
			})
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("internal error")
		}

		return &Output{
			Body: PostBooksOutputBody{
				ID: id,
			},
		}, nil
	})
}
