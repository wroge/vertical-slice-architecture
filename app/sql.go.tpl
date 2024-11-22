{{ define "filter_books" }}
    {{ if Postgres }}
        POSITION({{ . }} IN LOWER(books.title)) > 0
    {{ else }} 
        INSTR(LOWER(books.title), {{ . }}) 
    {{ end }}
    OR EXISTS (
        SELECT 1 FROM book_authors JOIN authors ON authors.id = book_authors.author_id
        WHERE book_authors.book_id = books.id
        AND (
            {{ if Postgres }} 
                POSITION({{ . }} IN LOWER(authors.name)) > 0
            {{ else }} 
                INSTR(LOWER(authors.name), {{ . }}) 
            {{ end }}
        )
    )
{{ end }}

{{ define "query_books" }}
    WITH filtered_books AS (
        SELECT books.id, books.title, books.number_of_pages
            {{ if Postgres }}
                , to_char(books.published_at, 'YYYY-MM-DD') AS published_at
                , jsonb_agg(jsonb_build_object('id', authors.id, 'name', authors.name)) AS authors
            {{ else }}
                , strftime('%Y-%m-%d', books.published_at) AS published_at
                , json_group_array(json_object('id', authors.id, 'name', authors.name)) AS authors
            {{ end }} 
        FROM books
        LEFT JOIN book_authors ON book_authors.book_id = books.id
        LEFT JOIN authors ON authors.id = book_authors.author_id
        {{ if .Search }} 
            WHERE {{ template "filter_books" (lower .Search) }}
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
        {{ if Postgres }}
            {{ ScanBooks Dest.Books ", jsonb_agg(jsonb_build_object('id', paginated_books.id, 'title', paginated_books.title, 'number_of_pages', paginated_books.number_of_pages, 'published_at', paginated_books.published_at, 'authors', paginated_books.authors))" }} 
        {{ else }}
            {{ ScanBooks Dest.Books ", json_group_array(json_object('id', paginated_books.id, 'title', paginated_books.title, 'number_of_pages', paginated_books.number_of_pages, 'published_at', paginated_books.published_at, 'authors', json(paginated_books.authors)))" }} 
        {{ end }}
    FROM paginated_books;
{{ end }}

{{ define "insert_authors" }}
    INSERT INTO authors (id, name) VALUES
    {{ range $i, $a := . }} {{ if $i }}, {{ end }}
        ({{ uuidv4 }}, {{ $a }})
    {{ end }}
    ON CONFLICT (name) DO NOTHING;
{{ end }}

{{ define "query_authors" }}
    SELECT id FROM authors WHERE name IN(
    {{ range $i, $a := . }} {{ if $i }}, {{ end }}
        {{ $a }}
    {{ end }});
{{ end }}

{{ define "insert_book"}}
    INSERT INTO books (id, title, published_at, number_of_pages) VALUES
        ({{ uuidv4 }},{{ .Title }},{{ .PublishedAt }}, {{ .NumberOfPages }})
    RETURNING id;
{{ end }}

{{ define "insert_book_authors" }}
    INSERT INTO book_authors (book_id, author_id) VALUES
    {{ range $i, $a := .AuthorIDs }} {{ if $i }}, {{ end }}
        ({{ $.BookID }}, {{ $a }})
    {{ end }};
{{ end }}