package app

import (
	"context"
	"net/http"
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
	query := sqlt.QueryStmt[*GetBooksInput, GetBooksOutputBody](a.Config, sqlt.ParseFiles("app/sql.go.tpl"), sqlt.Lookup("query_books"))

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
