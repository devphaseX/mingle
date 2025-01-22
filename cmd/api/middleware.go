package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/devphaseX/mingle.git/internal/store"
)

type authContext string

var (
	authKey authContext = "auth"
)

func (app *application) AuthTokenMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Vary", "Authorization")
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				app.authenticationRequiredResponse(w, r, "authorization header is missing or empty")
				return
			}

			parts := strings.Split(authHeader, " ")

			if !(len(parts) == 2 && parts[0] == "Bearer") {
				app.authenticationRequiredResponse(w, r, "authorization header is malformed")
				return
			}

			payload, err := app.tokenMaker.ValidateAccessToken(string(parts[1]))

			if err != nil {
				app.authenticationRequiredResponse(w, r, "invalid authenication token")
				return
			}

			fmt.Printf("%+v\n", payload)

			if err := payload.Valid(); err != nil {
				app.authenticationRequiredResponse(w, r, err.Error())
				return
			}

			user, err := app.store.Users.GetById(r.Context(), payload.UserID)

			if err != nil {
				app.authenticationRequiredResponse(w, r, err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), authKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (app *application) checkPostOwnership(role string, next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := getAuthUserFromCtx(r)
		post := getPostFromCtx(r)

		if post.UserID == user.ID {
			next.ServeHTTP(w, r)
			return
		}

		allow, err := app.checkRolePrecedence(r.Context(), user, role)

		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		if !allow {
			app.forbiddenErrorResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *application) checkRolePrecedence(ctx context.Context, user *store.User, roleName string) (bool, error) {
	role, err := app.store.Roles.GetByName(ctx, roleName)

	if err != nil {
		return false, nil
	}

	return user.Role.Level >= role.Level, nil
}

func getAuthUserFromCtx(r *http.Request) *store.User {
	user, ok := r.Context().Value(authKey).(*store.User)

	if !ok {
		panic("user context middleware not ran or functioning properly")
	}

	return user
}

func (app *application) BasicAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				app.authenticationBasicRequiredResponse(w, r, "authorization header is missing or empty")
				return
			}

			parts := strings.Split(authHeader, " ")

			if !(len(parts) == 2 && parts[0] == "Basic") {
				app.authenticationBasicRequiredResponse(w, r, "authorization header is malformed")
				return
			}

			decodedToken, err := base64.StdEncoding.DecodeString(parts[1])

			if err != nil {
				app.authenticationBasicRequiredResponse(w, r, err.Error())
				return
			}

			creds := bytes.SplitN(decodedToken, []byte(":"), 2)

			if !(len(creds) == 2 &&
				app.config.auth.basic.username == string(creds[0]) &&
				app.config.auth.basic.password == string(creds[1])) {
				app.authenticationBasicRequiredResponse(w, r, "invalid credentials username or password")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
