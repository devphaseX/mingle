package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/devphaseX/mingle.git/internal/validator"
)

func (app *application) errorResponse(
	w http.ResponseWriter, _ *http.Request, status int, message any) {

	env := envelope{"error": message}

	err := app.writeJSON(w, status, env, nil)

	if err != nil {
		w.WriteHeader(500)
	}
}

func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Errorw("interal server error", "method", r.Method, "path", r.URL.Path, "error", err)

	message := "the server encountered a problem and could not process your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (app *application) forbiddenErrorResponse(w http.ResponseWriter, r *http.Request) {
	app.logger.Warnw("forbidden error", "method", r.Method, "path", r.URL.Path)

	message := `You do not have permission to access this resource. Please contact your administrator if you believe this is an error.`
	app.errorResponse(w, r, http.StatusForbidden, message)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	app.logger.Errorf("not found error", "method", r.Method, "path", r.URL.Path)

	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnf("bad request error", "method", r.Method, "path", r.URL.Path, "error", err)

	var validationErrors *validator.ValidationErrors

	if errors.As(err, &validationErrors) {
		app.errorResponse(w, r, http.StatusBadRequest, validationErrors.FieldErrors())
		return
	}
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}

func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	app.logger.Errorf("conflict response", "method", r.Method, "path", r.URL.Path)

	message := "unable to update the record due to an edit conflict, please try again"
	app.errorResponse(w, r, http.StatusConflict, message)
}

func (app *application) conflictResponse(w http.ResponseWriter, r *http.Request, message string) {
	app.logger.Errorf("conflict response", "method", r.Method, "path", r.URL.Path)
	app.errorResponse(w, r, http.StatusConflict, message)
}

func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request, retryAfter string) {
	app.logger.Warnf("rate limit exceeded", "method", r.Method, "path", r.URL.Path)
	w.Header().Set("Retry-After", retryAfter)

	message := fmt.Sprintf("rate limit exceeded, retry after: %s", retryAfter)
	app.errorResponse(w, r, http.StatusTooManyRequests, message)
}

func (app *application) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (app *application) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request, message string) {
	app.logger.Errorf("unauthorized response", "method", r.Method, "path", r.URL.Path)
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (app *application) authenticationBasicRequiredResponse(w http.ResponseWriter, r *http.Request, message string) {
	app.logger.Errorf("unauthorized response", "method", r.Method, "path", r.URL.Path)
	w.Header().Set("WWW-Authenticate", `Basic realm="restricted",charset="UTF-8"`)
	app.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (app *application) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}

func (app *application) notPermittedResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account doesn't have the necessary permissions to access this resource"
	app.errorResponse(w, r, http.StatusForbidden, message)
}
