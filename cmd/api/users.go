package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/devphaseX/mingle.git/internal/store"
)

func (app *application) getUserByIdHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := app.readIntID(r, "userID")

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := context.Background()

	user, err := app.store.Users.GetById(ctx, userId)

	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
