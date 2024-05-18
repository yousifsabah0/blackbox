package main

import "net/http"

func (app *application) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"version": version,
		"status":  "available",
		"message": "ok",
	}

	if err := app.JSON(w, http.StatusOK, envelope{"data": response}); err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
