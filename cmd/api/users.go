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

// GetUser godoc
//
//	@Summary		Fetches a user profile
//	@Description	Fetch a user profile by id
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int						true	"User ID"
//	@Success		200	{object}	object{user=store.User}	"Success response with user data"
//	@Failure		400	{object}	object{error=string}	"Bad request"
//	@Failure		404	{object}	object{error=string}	"User not found"
//	@Failure		500	{object}	object{error=string}	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/users/{id} [get]
func (app *application) getUserByIdHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromCtx(r)
	err := app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// FollowUser godoc
//
//	@Summary		Follow a user
//	@Description	Follow a user by their ID. The follower ID is hardcoded to 1 for this example.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int										true	"User ID of the user to follow"
//	@Success		201	{object}	object{follower=store.Follower}			"Successfully followed the user"
//	@Failure		409	{object}	object{error=object{message=string}}	"Conflict - Already following this user"
//	@Failure		500	{object}	object{error=object{message=string}}	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/users/{id}/follow [put]
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
		case errors.Is(err, store.ErrConflict):
			app.errorResponse(w, r, http.StatusConflict, "following this user already")
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

// UnfollowUser godoc
//
//	@Summary		Unfollow a user
//	@Description	Unfollow a user by their ID. The follower ID is hardcoded to 1 for this example.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			id	path	int	true	"User ID of the user to unfollow"
//	@Success		204	"Successfully unfollowed the user"
//	@Failure		404	{object}	object{error=object{message=string}}	"Not Found - No follow relationship found"
//	@Failure		500	{object}	object{error=object{message=string}}	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/users/{id}/unfollow [put]
func (app *application) unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	unfollowedUser := getUserFromCtx(r)
	var userId int64 = 1

	ctx := context.Background()

	err := app.store.Followers.UnFollowUser(ctx, unfollowedUser.ID, userId)

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

		user, err := app.store.Users.GetById(r.Context(), userId)

		if err != nil {
			switch {
			case errors.Is(err, store.ErrNotFound):
				app.notFoundResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
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
