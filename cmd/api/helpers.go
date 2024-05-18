package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/yousifsabah0/blackbox/internal/validator"
)

type envelope map[string]any

const (
	jsonContentType = "application/json"
	maxBodyBytes    = 1_048_567
)

// ReadString returns a string value from the query string
func (app *application) ReadString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	return s
}

// ReadCSV returns a string value from the query string and splits
// them into a slice
func (app *application) ReadCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)
	if csv == "" {
		return defaultValue
	}

	return strings.Split(csv, ",")
}

// ReadInt reads a string from the query string and convert it into
// an integer
func (app *application) ReadInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddErrors(key, "must be an integer value")
		return defaultValue
	}

	return i
}

// ParseIDParams used to get the query parameters for the id
//
//	from the request
func (app *application) ParseIDParams(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// JSON is a helper method to write JSON responses
func (app *application) JSON(w http.ResponseWriter, status int, v envelope, headers ...http.Header) error {
	payload, err := json.Marshal(&v)
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("content-type", jsonContentType)
	w.WriteHeader(status)
	w.Write(payload)

	return nil
}

// Bind is a helper method to read JSON from the request
func (app *application) Bind(r *http.Request, v any) error {
	decoder := json.NewDecoder(r.Body)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&v); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed json (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("error must not be empty")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect json type for field %s", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect json type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed json")

		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			field := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", field)
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBodyBytes)
		default:
			return err
		}
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return err
	}

	return nil
}

func (app *application) background(fn func()) {
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Errorf("%s", err), nil)
			}
		}()
		fn()
	}()
}
