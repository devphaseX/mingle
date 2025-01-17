package store

import (
	"net/http"
	"strings"
	"time"
)

type Filterable interface {
	ParseFilters(r *http.Request) error
}

type GetUserFeedFilter struct {
	Search *string    `json:"search" validate:"omitempty,min=1"`
	Tags   []string   `json:"tags" validate:"omitempty"`
	Since  *time.Time `json:"since" validate:"omitempty"`
	Until  *time.Time `json:"until" validate:"omitempty,gtfield=Since"`
	Filterable
}

// ParseFilters extracts filter criteria from the HTTP request.
func (f *GetUserFeedFilter) ParseFilters(r *http.Request) error {
	qs := r.URL.Query()

	// Parse the "title" filter
	if search := qs.Get("search"); search != "" {
		f.Search = &search
	}

	if tags := qs.Get("tags"); tags != "" {
		f.Tags = strings.Split(tags, ",")
	}

	if since := qs.Get("since"); since != "" {
		if t, err := parseTime(since); err != nil {
			return err
		} else {
			f.Since = t
		}

	}

	if until := qs.Get("until"); until != "" {
		if t, err := parseTime(until); err != nil {
			return err
		} else {
			f.Until = t
		}
	}

	return nil
}

func parseTime(s string) (*time.Time, error) {
	t, err := time.Parse(time.DateTime, s)

	if err != nil {
		return nil, err
	}

	return &t, nil
}
