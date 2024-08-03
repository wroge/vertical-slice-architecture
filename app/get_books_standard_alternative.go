package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/danielgtaylor/huma/v2"
)

func (a *App) GetBooksStandardAlternative(api huma.API) {
	query := func(ctx context.Context, input *GetBooksInput) (GetBooksOutputBody, error) {
		var (
			query strings.Builder
			args  []any
		)

		query.WriteString(`
		WITH filtered_books AS (
			SELECT books.id, books.title, books.number_of_pages,`)

		if a.Dialect == "postgres" {
			query.WriteString(`
				to_char(published_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS published_at,
				json_agg(json_build_object('id', authors.id, 'name', authors.name)) AS authors`)
		} else {
			query.WriteString(`
				strftime('%Y-%m-%dT%H:%M:%SZ', published_at) AS published_at,
				json_group_array(json_object('id', authors.id, 'name', authors.name)) AS authors`)
		}

		query.WriteString(`
			FROM books
			LEFT JOIN book_authors ON book_authors.book_id = books.id
			LEFT JOIN authors ON authors.id = book_authors.author_id`)

		if input.Search != "" {
			query.WriteString(`
				WHERE (`)
			if a.Dialect == "postgres" {
				query.WriteString(` POSITION(? IN books.title) > 0`)
			} else {
				query.WriteString(` INSTR(books.title, ?) > 0`)
			}
			args = append(args, input.Search)
			query.WriteString(`
					OR books.id IN (
						SELECT book_authors.book_id
						FROM book_authors
						JOIN authors ON authors.id = book_authors.author_id
						WHERE`)
			if a.Dialect == "postgres" {
				query.WriteString(` POSITION(? IN authors.name) > 0`)
			} else {
				query.WriteString(` INSTR(authors.name, ?) > 0`)
			}
			args = append(args, input.Search)
			query.WriteString("))")
		}

		query.WriteString(`
			GROUP BY books.id, books.title, books.number_of_pages, books.published_at
		),
		paginated_books AS (
			SELECT id, title, number_of_pages, published_at, authors
			FROM filtered_books`)

		if input.Sort != "" {
			query.WriteString(`
				ORDER BY`)
			switch input.Sort {
			case "id":
				query.WriteString(` id`)
			case "title":
				query.WriteString(` title`)
			case "number_of_pages":
				query.WriteString(` number_of_pages`)
			case "published_at":
				query.WriteString(` published_at`)
			}
			if input.Direction == "desc" {
				query.WriteString(` DESC NULLS LAST`)
			} else {
				query.WriteString(` ASC NULLS LAST`)
			}
		}

		if input.Limit > 0 {
			query.WriteString(` LIMIT ?`)
			args = append(args, input.Limit)
		}
		if input.Offset > 0 {
			query.WriteString(` OFFSET ?`)
			args = append(args, input.Offset)
		}

		query.WriteString(`)
		SELECT
			(SELECT COUNT(*) FROM filtered_books),`)

		if a.Dialect == "postgres" {
			query.WriteString(`
				json_agg(json_build_object('id', id, 'title', title, 'number_of_pages', number_of_pages, 'published_at', published_at, 'authors', authors))`)
		} else {
			query.WriteString(`
				json_group_array(json_object('id', id, 'title', title, 'number_of_pages', number_of_pages, 'published_at', published_at, 'authors', json(authors)))`)
		}

		query.WriteString(`
			AS books
		FROM paginated_books;`)

		var (
			err   error
			body  GetBooksOutputBody
			books []byte
			str   = query.String()
		)

		if a.Dialect == "postgres" {
			str, err = squirrel.Dollar.ReplacePlaceholders(str)
			if err != nil {
				return body, err
			}
		}

		if err = a.DB.QueryRowContext(ctx, str, args...).Scan(&body.Total, &books); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return body, nil
			}

			return body, err
		}

		if err = json.Unmarshal(books, &body.Books); err != nil {
			return body, err
		}

		return body, nil
	}

	op := huma.Operation{
		Method:          http.MethodGet,
		Path:            "/standard_alternative/books",
		DefaultStatus:   http.StatusOK,
		MaxBodyBytes:    1 << 20, // 1MB
		BodyReadTimeout: time.Second / 2,
		Errors:          []int{http.StatusInternalServerError},
		Summary:         "Query Books Standard Alternative",
		Description:     "Query Books Standard Alternative",
	}

	huma.Register(api, op, func(ctx context.Context, input *GetBooksInput) (*GetBooksOutput, error) {
		body, err := query(ctx, input)
		if err != nil {
			a.Logger.Print(err)

			return nil, huma.Error500InternalServerError("internal error")
		}

		return &GetBooksOutput{
			Body: body,
		}, nil
	})
}
