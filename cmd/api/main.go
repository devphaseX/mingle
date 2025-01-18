package main

import (
	"log"

	"github.com/devphaseX/mingle.git/internal/db"
	"github.com/devphaseX/mingle.git/internal/env"
	"github.com/devphaseX/mingle.git/internal/store"

	_ "github.com/lib/pq"
)

var version = "0.0.2"

//	@title			Mingle Socials API
//	@version		0.0.1
//	@description	API FOR gopher social.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8080
//	@BasePath	/v1

// @securityDefinitions.apikey	Bearer
// @in							header
// @name						Authorization
// @description				Bearer token authentication
func main() {
	cfg := config{
		apiURL: env.GetString("EXTERNAL_LINKS", "localhost:8080"),
		addr:   env.GetString("ADDR", ":8080"),
		env:    env.GetString("ENV", "development"),
		db: dbConfig{
			dsn:          env.GetString("DB_ADDR", "postgres://mingle:adminpassword@localhost/mingle?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
	}

	db, err := db.New(cfg.db.dsn, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)

	defer db.Close()

	log.Println("database connection pool established")
	if err != nil {
		log.Panic(err)
	}

	store := store.NewPostgressStorage(db)
	app := &application{
		config: cfg,
		store:  store,
	}

	mux := app.mount()
	log.Fatal(app.serve(mux))
}
