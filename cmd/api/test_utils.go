package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devphaseX/mingle.git/internal/store"
	"github.com/devphaseX/mingle.git/internal/store/cache"
	"go.uber.org/zap"
)

func newTestApplication(t *testing.T) *application {
	t.Helper()

	// logger := zap.NewNop().Sugar()
	logger := zap.Must(zap.NewProduction()).Sugar()
	mockStore := store.NewMockStore()
	mockCacheStore := cache.NewMockCache()

	testAuth := &store.TestTokenStore{}
	return &application{
		logger:       logger,
		store:        mockStore,
		cacheStorage: mockCacheStore,
		tokenMaker:   testAuth,
	}
}

func executeRequest(req *http.Request, mux http.Handler) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d", expected, actual)
	}
}
