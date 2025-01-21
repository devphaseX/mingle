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

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

//	@Summary		Refresh access token
//	@Description	Refreshes an access token using a refresh token provided either in a cookie or in the request body.
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		refreshRequest																									false	"Refresh token (if not provided in cookie)"
//	@Success		200		{object}	object{access_token=string,access_token_expires_in=int64,refresh_token=string,refresh_token_expires_in=int64}	"Returns a new access token and optionally a new refresh token"
//	@Failure		400		{object}	object{error=string}																							"Invalid request payload"
//	@Failure		401		{object}	object{error=string}																							"Invalid refresh token or session"
//	@Failure		500		{object}	object{error=string}																							"Internal server error"
//	@Router			/auth/refresh [post]
func (app *application) refreshToken(w http.ResponseWriter, r *http.Request) {
	var form refreshRequest

	// Try to get the refresh token from the cookie
	refreshTokenCookie, err := r.Cookie("sid")
	if err == nil && refreshTokenCookie != nil && strings.TrimSpace(refreshTokenCookie.Value) != "" {
		form.RefreshToken = refreshTokenCookie.Value
	} else {
		// If the cookie is not available, try to get the refresh token from the JSON body
		if err := app.readJSON(w, r, &form); err != nil {
			app.errorResponse(w, r, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Validate the refresh token
	claims, err := app.tokenMaker.ValidateRefreshToken(form.RefreshToken)
	if err != nil {
		app.errorResponse(w, r, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	// Validate the session
	session, user, canExtend, err := app.store.Sessions.ValidateSession(r.Context(), claims.SessionID, claims.Version)

	if err != nil || session == nil {
		app.errorResponse(w, r, http.StatusUnauthorized, "invalid session")
		return
	}

	// Generate a new access token
	accessToken, err := app.tokenMaker.GenerateAccessToken(user.ID, session.ID, app.config.auth.AccessTokenTTL)
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
		"access_token":            accessToken,
		"access_token_expires_in": time.Now().Add(app.config.auth.AccessTokenTTL).Unix(),
	}
	if newRefreshToken != "" {
		response["refresh_token"] = newRefreshToken
		response["refresh_token_expires_in"] = time.Now().Add(app.config.auth.AccessTokenTTL).Unix()
	}

	if err := app.writeJSON(w, http.StatusOK, response, nil); err != nil {
		app.errorResponse(w, r, http.StatusInternalServerError, "failed to write response")
	}
}

type signInForm struct {
	Email      string `json:"email" validate:"required,email,max=255"`
	Password   string `json:"password" validate:"required,min=1,max=255"`
	RememberMe bool   `json:"remember_me"`
}

// signInForm godoc
//
//	@Summary		Sign in a user
//	@Description	Authenticates a user and returns access and refresh tokens.
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			body	body		signInForm	true	"Sign-in request body"
//	@Success		200		{object}	object{access_token=string,access_token_expires_in=int64,refresh_token=string,refresh_token_expires_in=int64}
//	@Failure		400		{object}	object{error=string}
//	@Failure		404		{object}	object{error=string}
//	@Failure		500		{object}	object{error=string}
//	@Router			/sign-in [post]
func (app *application) signInHandler(w http.ResponseWriter, r *http.Request) {
	var form signInForm
	if err := app.readJSON(w, r, &form); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.store.Users.GetByEmail(r.Context(), form.Email)

	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.errorResponse(w, r, http.StatusNotFound, "invalid credential email or password")
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	match, err := user.Password.Matches(form.Password)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if !match {
		app.errorResponse(w, r, http.StatusNotFound, "invalid credential email or password")
		return
	}

	sessionExpiry := app.config.auth.RefreshTokenTTL
	if form.RememberMe {
		sessionExpiry = app.config.auth.RememberMeTTL
	}

	session, err := app.store.Sessions.CreateSession(r.Context(), user.ID, "", "", sessionExpiry, form.RememberMe)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	accessToken, err := app.tokenMaker.GenerateAccessToken(user.ID, session.ID, app.config.auth.AccessTokenTTL)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	refreshToken, err := app.tokenMaker.GenerateRefreshToken(session.ID, session.Version, sessionExpiry)

	response := envelope{
		"access_token":             accessToken,
		"access_token_expires_in":  time.Now().Add(app.config.auth.AccessTokenTTL).Unix(),
		"refresh_token":            refreshToken, // Include the new refresh token (if generated)
		"refresh_token_expires_in": sessionExpiry,
	}

	if err := app.writeJSON(w, http.StatusOK, response, nil); err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
