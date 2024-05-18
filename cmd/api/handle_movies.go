package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/yousifsabah0/blackbox/internal/data"
	"github.com/yousifsabah0/blackbox/internal/validator"
)

func (app *application) handleCreateMovie(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	if err := app.Bind(r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	movie := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	v := validator.New()
	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.Movie.Insert(movie); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))

	if err := app.JSON(w, http.StatusCreated, envelope{"movie": movie}, headers); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) handleShowAllMovies(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string
		Genres []string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Title = app.ReadString(qs, "title", "")
	input.Genres = app.ReadCSV(qs, "genres", []string{})

	input.Filters.Page = app.ReadInt(qs, "page", 1, v)
	input.Filters.PageSize = app.ReadInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.ReadString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	movies, metadata, err := app.models.Movie.SelectMany(input.Title, input.Genres, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.JSON(w, http.StatusOK, envelope{"movies": movies, "metadata": metadata}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) handleShowMovie(w http.ResponseWriter, r *http.Request) {
	id, err := app.ParseIDParams(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.models.Movie.Select(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.JSON(w, http.StatusOK, envelope{"movie": movie}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) handleUpdateMovie(w http.ResponseWriter, r *http.Request) {
	id, err := app.ParseIDParams(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.models.Movie.Select(id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	var input struct {
		Title   *string       `json:"title"`
		Year    *int32        `json:"year"`
		Runtime *data.Runtime `json:"runtime"`
		Genres  []string      `json:"genres"`
	}

	if err := app.Bind(r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Title != nil {
		movie.Title = *input.Title
	}

	if input.Year != nil {
		movie.Year = *input.Year
	}

	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}

	if input.Genres != nil {
		movie.Genres = input.Genres
	}

	v := validator.New()
	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.Movie.Update(movie); err != nil {
		if errors.Is(err, data.ErrEditConflict) {
			app.editConflictResponse(w, r)
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.JSON(w, http.StatusOK, envelope{"movie": movie}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) handleDeleteMovie(w http.ResponseWriter, r *http.Request) {
	id, err := app.ParseIDParams(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	if err := app.models.Movie.Delete(id); err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.JSON(w, http.StatusOK, envelope{"message": "deleted"}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
