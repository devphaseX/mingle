package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/devphaseX/mingle.git/internal/store"
	"github.com/google/uuid"
)

type registerUserForm struct {
	FirstName string `json:"first_name" validate:"min=1,max=255"`
	LastName  string `json:"last_name" validate:"min=1,max=255"`
	Username  string `json:"username" validate:"min=1,max=255"`
	Email     string `json:"email" validate:"email,min=1,max=255"`
	Password  string `json:"password" validate:"min=8,max=50"`
}

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var form registerUserForm

	if err := app.readJSON(w, r, &form); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(form); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &store.User{
		FirstName: form.FirstName,
		LastName:  form.LastName,
		Username:  form.Username,
		Email:     form.Email,
	}

	user.Password.Set(form.Password)

	plainToken := uuid.New().String()

	fmt.Println("token", plainToken)
	//store
	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])

	err := app.store.Users.CreateAndInvite(r.Context(), user, app.config.mail.exp, hashToken)

	if err != nil {
		var friendlyError store.UserFriendlyError
		if errors.As(err, &friendlyError) {
			switch {
			case errors.Is(friendlyError.InternalErr, store.ErrConflict):
				app.conflictResponse(w, r, friendlyError.UserMessage)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusCreated, nil, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
