package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/devphaseX/mingle.git/internal/store"
)

type postContextKey string

var (
	postCtxKey postContextKey = "post"
)

type createPostForm struct {
	Title   string   `json:"title" validate:"required,max=100"`
	Content string   `json:"content" validate:"required,max=1000"`
	Tags    []string `json:"tags"`
}

// CreatePost godoc
//
//	@Summary		Creates a post
//	@Description	Creates a post
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		createPostForm	true	"Post payload"
//	@Success		201		{object}	object{post=store.Post}
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/posts [post]
func (app *application) createPostHandler(w http.ResponseWriter, r *http.Request) {
	var (
		form createPostForm
		err  error
	)

	if err = app.readJSON(w, r, &form); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(form); err != nil {
		app.failedValidationResponse(w, r, err.FieldErrors())
		return
	}

	user := getAuthUserFromCtx(r)
	post := &store.Post{
		Title:   form.Title,
		Context: form.Content,
		Tags:    form.Tags,
		UserID:  user.ID,
	}

	ctx := context.Background()
	err = app.store.Posts.Create(ctx, post)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"post": post}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// GetPostByID godoc
//
//	@Summary		Get a post by ID
//	@Description	Fetch a post by its ID, including its associated comments.
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int						true	"Post ID"
//	@Success		200	{object}	object{post=store.Post}	"Successfully fetched post with comments"
//	@Failure		404	{object}	object{error=string}	"Not Found - Post not found"
//	@Failure		500	{object}	object{error=string}	"Internal server error"
//	@Security		ApiKeyAuth
//	@Router			/posts/{id} [get]
func (app *application) getPostByIdHandler(w http.ResponseWriter, r *http.Request) {
	post := getPostFromCtx(r)
	comments, err := app.store.Comments.GetByPostID(r.Context(), post.ID)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	post.Comments = comments
	err = app.writeJSON(w, http.StatusOK, envelope{"post": post}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

// DeletePost godoc
//
//	@Summary		Deletes a post
//	@Description	Delete a post by ID
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Post ID"
//	@Success		204	{object}	string
//	@Failure		404	{object}	error
//	@Failure		500	{object}	error
//	@Security		ApiKeyAuth
//	@Router			/posts/{id} [delete]
func (app *application) removePostByIdHandler(w http.ResponseWriter, r *http.Request) {
	var (
		post = getPostFromCtx(r)
		user = getAuthUserFromCtx(r)
	)

	ctx := context.Background()
	err := app.store.Posts.DeleteByUser(ctx, post.ID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r)

		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type updatePostForm struct {
	Title   *string `json:"title" validate:"omitempty,max=100"`
	Content *string `json:"content" validate:"omitempty,max=100"`
}

// UpdatePost godoc
//
//	@Summary		Updates a post
//	@Description	Updates a post by ID
//	@Tags			posts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Post ID"
//	@Param			payload	body		updatePostForm	true	"Post data"
//	@Success		200		{object}	object{follower=store.Post}
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		404		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/posts/{id} [patch]
func (app *application) updatePostHandler(w http.ResponseWriter, r *http.Request) {
	post := getPostFromCtx(r)

	var form updatePostForm
	err := app.readJSON(w, r, &form)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(form); err != nil {
		app.failedValidationResponse(w, r, err.FieldErrors())
		return
	}

	if form.Title != nil {
		post.Title = *form.Title
	}

	if form.Content != nil {
		post.Context = *form.Content
	}

	ctx := context.Background()

	err = app.store.Posts.UpdateByUser(ctx, post)

	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.errorResponse(w, r, http.StatusConflict, nil)
		default:
			app.serverErrorResponse(w, r, err)

		}
		return
	}
	app.writeJSON(w, http.StatusOK, envelope{"post": post}, nil)
}

func (app *application) postContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		postID, err := app.readIntID(r, "postID")

		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}

		post, err := app.store.Posts.GetById(r.Context(), postID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				app.notFoundResponse(w, r)
				return
			}
			app.serverErrorResponse(w, r, err)
			return
		}

		fmt.Println(post)

		ctx := context.WithValue(r.Context(), postCtxKey, post)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func getPostFromCtx(r *http.Request) *store.Post {
	post, _ := r.Context().Value(postCtxKey).(*store.Post)

	return post
}
