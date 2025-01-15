package main

import (
	"log"

	"github.com/devphaseX/mingle.git/internal/env"
	"github.com/devphaseX/mingle.git/internal/store"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config{
		addr: env.GetString("ADDR", ":8080"),
	}

	store := store.NewPostgressStorage(nil)

	app := &application{
		config: cfg,
		store:  store,
	}

	mux := app.mount()
	log.Fatal(app.serve(mux))
}
