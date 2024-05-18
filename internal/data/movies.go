package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/yousifsabah0/blackbox/internal/validator"
)

type Movie struct {
	ID int64 `json:"id"`

	Title string `json:"title"`
	Year  int32  `json:"year"`

	Runtime Runtime  `json:"runtime"`
	Genres  []string `json:"genres"`

	Version   int32     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

type MovieModel struct {
	DB *sql.DB
}

const (
	timeout = 10 * time.Second
)

func (m MovieModel) Insert(movie *Movie) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `
						INSERT INTO movies
						(title, year, runtime, genres)
						VALUES 
						($1, $2, $3, $4)
						RETURNING id, created_at, version
	`
	args := []any{movie.Title, movie.Year, movie.Runtime, pq.Array(movie.Genres)}

	row := m.DB.QueryRowContext(ctx, query, args...)
	if err := row.Scan(&movie.ID, &movie.CreatedAt, &movie.Version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}

		return err
	}

	return nil
}

func (m MovieModel) Select(id int64) (*Movie, error) {
	var movie Movie

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `
						SELECT 
						id, title, year, runtime, genres, version, created_at 
						FROM movies 
						WHERE 
						id = $1
	`

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&movie.ID,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
		&movie.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &movie, nil
}

func (m MovieModel) SelectMany(title string, genres []string, filters Filters) ([]*Movie, MetaData, error) {
	total := 0
	var movies []*Movie
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := fmt.Sprintf(`
				SELECT
				count(*) OVER(), id, title, year, runtime, genres, version, created_at
				FROM movies
				WHERE
				(to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
				AND
				(genres @> $2 OR $2 = '{}')
				ORDER BY %s %s, id ASC
				LIMIT $3
				OFFSET $4
	`, filters.sortColumn(), filters.sortDirection())
	args := []any{title, pq.Array(genres), filters.limit(), filters.offset()}
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, MetaData{}, err
	}

	defer rows.Close()

	for rows.Next() {
		var movie Movie
		err := rows.Scan(
			&total,
			&movie.ID,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			pq.Array(&movie.Genres),
			&movie.Version,
			&movie.CreatedAt,
		)
		if err != nil {
			return nil, MetaData{}, err
		}

		movies = append(movies, &movie)
	}

	if err := rows.Err(); err != nil {
		return nil, MetaData{}, err
	}

	metadata := calculateMetaData(total, filters.Page, filters.PageSize)

	return movies, metadata, nil
}

func (m MovieModel) Update(movie *Movie) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `
					UPDATE movies
					SET
					title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
					WHERE
					version = $5 
					AND
					id = $6
					RETURNING version
	`
	args := []any{&movie.Title, &movie.Year, &movie.Runtime, pq.Array(&movie.Genres), &movie.Version, &movie.ID}

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEditConflict
		}
		return err
	}

	return nil
}

func (m MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `
					DELETE 
					FROM movies
					WHERE
					id = $1
	`

	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(len(movie.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(movie.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(movie.Genres) <= 5, "genres", "must not contain more than 5 genres")

	v.Check(validator.Unique(movie.Genres), "genres", "must not contain duplicate values")

}
