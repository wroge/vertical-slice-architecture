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

		InsertBookAuthor struct {
			BookID    uuid.UUID
			AuthorIDs []uuid.UUID
		}
	)

	insertAuthors := sqlt.Stmt[[]string](a.Config, sqlt.ParseFiles("app/sql.go.tpl"), sqlt.Lookup("insert_authors"))

	queryAuthors := sqlt.QueryStmt[[]string, uuid.UUID](a.Config, sqlt.ParseFiles("app/sql.go.tpl"), sqlt.Lookup("query_authors"))

	insertBook := sqlt.QueryStmt[PostBooksInputBody, uuid.UUID](a.Config, sqlt.ParseFiles("app/sql.go.tpl"), sqlt.Lookup("insert_book"))

	insertBookAuthors := sqlt.Stmt[InsertBookAuthor](a.Config, sqlt.ParseFiles("app/sql.go.tpl"), sqlt.Lookup("insert_book_authors"))

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
