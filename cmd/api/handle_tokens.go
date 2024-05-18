package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/yousifsabah0/blackbox/internal/data"
	"github.com/yousifsabah0/blackbox/internal/validator"
)

func (app *application) handleCreateAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := app.Bind(r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordText(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.User.GetByEmail(input.Email)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.invalidCredentialResponse(w, r)
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if !match {
		app.invalidCredentialResponse(w, r)
		return
	}

	token, err := app.models.Token.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.JSON(w, http.StatusOK, envelope{"token": token}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) handleResendActivationToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}

	if err := app.Bind(r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidateEmail(v, input.Email); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.User.GetByEmail(input.Email)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			v.AddErrors("email", "no matching email found")
			app.failedValidationResponse(w, r, v.Errors)
			return
		}
	}

	if user.Activated {
		v.AddErrors("email", "email has already been activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := app.models.Token.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]any{
			"activationToken": token.Text,
		}

		if err := app.mailer.Send(user.Email, "token_activation.html", data); err != nil {
			app.logger.Error(err, nil)
		}
	})

	if err := app.JSON(w, http.StatusAccepted, envelope{"message": "check your email1"}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) handleRequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}

	if err := app.Bind(r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidateEmail(v, input.Email); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.User.GetByEmail(input.Email)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			v.AddErrors("email", "no matching email found")
			app.failedValidationResponse(w, r, v.Errors)
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	if !user.Activated {
		v.AddErrors("email", "account must be activated")
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	token, err := app.models.Token.New(user.ID, 45*time.Minute, data.ScopePasswordReset)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {
		data := map[string]any{
			"passwordResetToken": token.Text,
		}

		if err := app.mailer.Send(user.Email, "password_reset.html", data); err != nil {
			app.logger.Error(err, nil)
		}
	})

	if err := app.JSON(w, http.StatusAccepted, envelope{"message": "check your email1"}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) handleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Password string `json:"password"`
		Token    string `json:"token"`
	}

	if err := app.Bind(r, &input); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()

	data.ValidatePasswordText(v, input.Password)
	data.ValidTokenText(v, input.Token)

	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.models.User.GetForToken(data.ScopePasswordReset, input.Token)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			v.AddErrors("token", "invalid token or expired")
			app.failedValidationResponse(w, r, v.Errors)
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	if err := user.Password.Hash(input.Password); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.models.User.Update(user); err != nil {
		if errors.Is(err, data.ErrEditConflict) {
			app.editConflictResponse(w, r)
			return
		}

		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.models.Token.DeleteAllForUser(data.ScopePasswordReset, user.ID); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.JSON(w, http.StatusOK, envelope{"message": "password updated"}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
