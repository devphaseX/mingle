package main

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (app *application) readIntID(r *http.Request, param string) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)

	if err != nil {
		return 0, err
	}

	return id, nil
}
