package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/danielgtaylor/huma/v2"
)

func (a *App) GetBooksStandardAlternative(api huma.API) {
	filter := func(search string) (string, []any) {
		if a.Dialect == Postgres {
			return `POSITION(? IN LOWER(books.title)) > 0
			EXISTS (
				SELECT 1 FROM book_authors
				JOIN authors ON authors.id = book_authors.author_id
				WHERE book_authors.book_id = books.id
				AND authors.name ILIKE '%' || ? || '%'
			)`, []any{search, search}
		} else {
			return `INSTR(LOWER(books.title), ?)  OR
			EXISTS (
				SELECT 1 FROM book_authors
				JOIN authors ON authors.id = book_authors.author_id
				WHERE book_authors.book_id = books.id
				AND authors.name LIKE '%' || ? || '%'
			)`, []any{search, search}
		}
	}

	query := func(ctx context.Context, input *GetBooksInput) (GetBooksOutputBody, error) {
		filtered_books := squirrel.Select(
			"books.id",
			"books.title",
			"books.number_of_pages",
		).From("books").
			LeftJoin("book_authors ON book_authors.book_id = books.id").
			LeftJoin("authors ON authors.id = book_authors.author_id").
			GroupBy(
				"books.id",
				"books.title",
				"books.number_of_pages",
				"books.published_at",
			)

		if a.Dialect == Postgres {
			filtered_books = filtered_books.Columns(
				`to_char(published_at, 'YYYY-MM-DD') AS published_at`,
				"jsonb_agg(jsonb_build_object('id', authors.id, 'name', authors.name)) AS authors",
			)
		} else {
			filtered_books = filtered_books.Columns(
				`strftime('%Y-%m-%d', published_at) AS published_at`,
				"json_group_array(json_object('id', authors.id, 'name', authors.name)) AS authors",
			)
		}
		if input.Search != "" {
			searchQuery, fArgs := filter(input.Search)
			filtered_books = filtered_books.Where(searchQuery, fArgs...)
		}

		paginated_books := squirrel.Select("id", "title", "number_of_pages", "published_at", "authors").From("filtered_books")

		if input.Sort != "" {
			paginated_books = paginated_books.OrderBy(fmt.Sprintf("%s %s NULLS LAST", input.Sort, input.Direction))
		}

		if input.Limit > 0 {
			paginated_books = paginated_books.Limit(input.Limit)
		}

		if input.Offset > 0 {
			paginated_books = paginated_books.Offset(input.Offset)
		}

		queryBuilder := squirrel.Select("(SELECT COUNT(*) FROM filtered_books)").
			From("paginated_books").
			Prefix("WITH filtered_books AS (?), paginated_books AS (?)", filtered_books, paginated_books)

		if a.Dialect == Postgres {
			queryBuilder = queryBuilder.Column("jsonb_agg(jsonb_build_object('id', id, 'title', title, 'number_of_pages', number_of_pages, 'published_at', published_at, 'authors', authors))")
		} else {
			queryBuilder = queryBuilder.Column("json_group_array(json_object('id', id, 'title', title, 'number_of_pages', number_of_pages, 'published_at', published_at, 'authors', json(authors)))")
		}

		var (
			err   error
			body  GetBooksOutputBody
			books []byte
		)

		if a.Dialect == Postgres {
			queryBuilder = queryBuilder.PlaceholderFormat(squirrel.Dollar)
		}

		if err = queryBuilder.RunWith(a.DB).ScanContext(ctx, &body.Total, &books); err != nil {
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
			a.Logger.Error(err.Error())

			return nil, huma.Error500InternalServerError("internal error")
		}

		return &GetBooksOutput{
			Body: body,
		}, nil
	})
}
