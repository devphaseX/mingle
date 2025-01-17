package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/devphaseX/mingle.git/internal/store"
)

type userKey string

var (
	userContextKey = userKey("user")
)

func (app *application) getUserByIdHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromCtx(r)
	err := app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) followUserHandler(w http.ResponseWriter, r *http.Request) {
	followedUser := getUserFromCtx(r)
	var userId int64 = 1

	ctx := context.Background()

	follower := &store.Follower{
		UserID:     followedUser.ID,
		FollowerID: userId,
	}

	err := app.store.Followers.FollowUser(ctx, follower)

	if err != nil {
		switch {
		case errors.Is(err, store.ErrUserAlreadyFollowed):
			app.errorResponse(w, r, http.StatusConflict, err.Error())

		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"follower": follower}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	followedUser := getUserFromCtx(r)
	var userId int64 = 1

	ctx := context.Background()

	err := app.store.Followers.UnFollowUser(ctx, followedUser.ID, userId)

	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}

		return
	}
	err = app.writeJSON(w, http.StatusNoContent, nil, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) userContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		ctx = context.WithValue(ctx, userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserFromCtx(r *http.Request) *store.User {
	user, ok := r.Context().Value(userContextKey).(*store.User)

	if !ok {
		panic("user context middleware not ran or functioning properly")
	}

	return user
}
