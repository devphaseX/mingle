package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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

func (app *application) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var refreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}

	// Try to get the refresh token from the cookie
	refreshTokenCookie, err := r.Cookie("sid")
	if err == nil && refreshTokenCookie != nil && strings.TrimSpace(refreshTokenCookie.Value) != "" {
		refreshRequest.RefreshToken = refreshTokenCookie.Value
	} else {
		// If the cookie is not available, try to get the refresh token from the JSON body
		if err := app.readJSON(w, r, &refreshRequest); err != nil {
			app.errorResponse(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Validate the refresh token
	claims, err := app.tokenMaker.ValidateRefreshToken(refreshRequest.RefreshToken)
	if err != nil {
		app.errorResponse(w, r, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	// Validate the session
	session, user, canExtend, err := app.store.Sessions.ValidateSession(r.Context(), claims.SessionID)
	if err != nil || session == nil {
		app.errorResponse(w, r, http.StatusUnauthorized, "invalid session")
		return
	}

	// Generate a new access token
	accessToken, err := app.tokenMaker.GenerateAccessToken(user.ID, session.ID)
	if err != nil {
		app.errorResponse(w, r, http.StatusInternalServerError, "failed to generate access token")
		return
	}

	var (
		newRefreshToken string
		rememberPeriod  = app.config.auth.RefreshTokenTTL
	)

	if canExtend {
		newRefreshToken, err = app.store.Sessions.ExtendSessionAndGenerateRefreshToken(r.Context(), session, app.tokenMaker, rememberPeriod)
		if err != nil {
			app.errorResponse(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to extend session: %v", err))
			return
		}
	}

	// Return the new access token
	response := envelope{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    time.Now().Add(rememberPeriod).Unix(),
	}

	if err := app.writeJSON(w, http.StatusOK, response, nil); err != nil {
		app.errorResponse(w, r, http.StatusInternalServerError, "failed to write response")
	}
}
