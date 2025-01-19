package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/devphaseX/mingle.git/internal/mailer"
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

	activationURL := fmt.Sprintf("%s/confirm/%s", app.config.frontendURL, plainToken)

	vars := struct {
		Email         string
		Username      string
		ActivationURL string
	}{
		Email:         user.Email,
		Username:      fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		ActivationURL: activationURL,
	}

	//send mail

	go func() {
		err := app.mailer.Send(
			mailer.UserWelcomeTemplate,
			vars.Username,
			vars.Email,
			vars,
			app.config.env == "development",
		)

		if err != nil {
			app.logger.Errorw("error sending welcome email", "error", err)

			ctx := context.Background()
			//rollback user creation if email fails (SAGA pattern)
			if err := app.store.Users.Delete(ctx, user.ID); err != nil {
				app.logger.Errorw("error deleting user", "error", err)
			}
			return
		}
	}()

	if err := app.writeJSON(w, http.StatusCreated, nil, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
