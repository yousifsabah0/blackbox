package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/yousifsabah0/blackbox/internal/data"
	"github.com/yousifsabah0/blackbox/internal/validator"
)

func (app *application) handleUserRegister(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := app.Bind(r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	if err := user.Password.Hash(input.Password); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	if err := app.models.User.Insert(user); err != nil {
		if errors.Is(err, data.ErrDuplicateEmail) {
			v.AddErrors("email", "duplicate email, use another one")
			app.failedValidationResponse(w, r, v.Errors)
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.models.Permission.GrantUser(user.ID, "movies:read"); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	token, err := app.models.Token.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]any{
			"activationToken": token.Text,
			"userID":          user.ID,
		}
		if err := app.mailer.Send(user.Email, "welcome.html", data); err != nil {
			app.logger.Error(err, nil)
			return
		}
	})

	if err := app.JSON(w, http.StatusAccepted, envelope{"user": user}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) handleActivateUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TextToken string `json:"token"`
	}

	if err := app.Bind(r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidTokenText(v, input.TextToken); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.User.GetForToken(data.ScopeActivation, input.TextToken)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			v.AddErrors("token", "invalid or expired token")
			app.failedValidationResponse(w, r, v.Errors)
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	user.Activated = true
	if err := app.models.User.Update(user); err != nil {
		if errors.Is(err, data.ErrEditConflict) {
			app.editConflictResponse(w, r)
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.models.Token.DeleteAllForUser(data.ScopeActivation, user.ID); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.models.Permission.GrantUser(user.ID, "movies:write"); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.JSON(w, http.StatusOK, envelope{"user": user}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
