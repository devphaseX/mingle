package main

import (
	"net/http"

	"github.com/devphaseX/mingle.git/internal/store"
)

func (app *application) getUserFeedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fq := &store.PaginateQueryFilter{
		Page:         1,
		PageSize:     20,
		Sort:         "created_at",
		SortSafelist: []string{"created_at", "-created_at"},
	}

	if err := fq.Parse(r); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	var userId int64 = 1
	posts, metadata, err := app.store.Posts.GetUserFeed(ctx, userId, *fq)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"posts": posts, "metadata": metadata}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
