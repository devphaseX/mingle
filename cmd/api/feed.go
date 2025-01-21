package main

import (
	"net/http"

	"github.com/devphaseX/mingle.git/internal/store"
)

// getUserFeedHandler godoc
//
//	@Summary		Fetches the user feed
//	@Description	Fetches the user feed with pagination and filtering
//	@Tags			feed
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			page_size	query		int		false	"Number of items per page (default: 20)"
//	@Param			sort		query		string	false	"Sort order (e.g., 'created_at' or '-created_at')"
//	@Param			search		query		string	false	"Search term"
//	@Param			tags		query		string	false	"Comma-separated list of tags to filter by"
//	@Param			since		query		string	false	"Filter posts created after this timestamp (RFC3339 format)"
//	@Param			until		query		string	false	"Filter posts created before this timestamp (RFC3339 format)"
//	@Success		200			{object}	object{posts=[]store.PostWithMetadata, metadata=store.Metadata}
//	@Failure		400			{object}	error
//	@Failure		500			{object}	error
//	@Security		ApiKeyAuth
//	@Router			/users/feed [get]
func (app *application) getUserFeedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filter := &store.GetUserFeedFilter{}

	fq := &store.PaginateQueryFilter{
		Page:         1,
		PageSize:     20,
		Sort:         "created_at",
		SortSafelist: []string{"created_at", "-created_at"},
		Filters:      filter,
	}

	if err := fq.Parse(r); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := getAuthUserFromCtx(r)
	posts, metadata, err := app.store.Posts.GetUserFeed(ctx, user.ID, *fq)

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"posts": posts, "metadata": metadata}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
