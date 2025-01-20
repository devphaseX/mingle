package main

import (
	"log"
	"time"

	"github.com/devphaseX/mingle.git/internal/db"
	"github.com/devphaseX/mingle.git/internal/env"
	"github.com/devphaseX/mingle.git/internal/mailer"
	"github.com/devphaseX/mingle.git/internal/store"
	"go.uber.org/zap"

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
		apiURL:      env.GetString("EXTERNAL_LINKS", "localhost:8080"),
		addr:        env.GetString("ADDR", ":8080"),
		env:         env.GetString("ENV", "development"),
		frontendURL: env.GetString("FRONTEND_URL", "http://localhost:5173"),
		db: dbConfig{
			dsn:          env.GetString("DB_ADDR", "postgres://mingle:adminpassword@localhost/mingle?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
			maxIdleTime:  env.GetString("DB_MAX_IDLE_TIME", "15m"),
		},
		mail: mailConfig{
			exp: time.Hour * 24 * 3, //3 days
			mailTrap: mailTrapConfig{
				fromEmail:       env.GetString("MAIL_TRAP_FROM_EMAIL", ""),
				apiKey:          env.GetString("MAIL_TRAP_API_KEY", ""),
				smtpAddr:        env.GetString("MAIL_TRAP_SMTP_ADDR", ""),
				smtpSandboxAddr: env.GetString("MAIL_TRAP_SANDBOX_ADDR", "sandbox.smtp.mailtrap.io"),
				smtpPort:        env.GetInt("MAIL_TRAP_SMTP_PORT", 0),
				username:        env.GetString("MAIL_TRAP_USERNAME", ""),
				password:        env.GetString("MAIL_TRAP_PASSWORD", ""),
			},
		},

		auth: AuthConfig{
			AccessSecretKey:  env.GetString("ACCESS_SECRET_KEY", ""),
			RefreshSecretKey: env.GetString("REFRESH_SECRET_KEY", ""),
			AccessTokenTTL:   env.GetDuration("ACCESS_TOKEN_TTL", time.Minute*5),
			RefreshTokenTTL:  env.GetDuration("REFRESH_TOKEN_TLL", time.Hour*1),
			RememberMeTTL:    env.GetDuration("REMEMBER_ME_TTL", time.Hour*24*30),
			basic: basicAuth{
				username: env.GetString("AUTH_BASIC_USERNAME", ""),
				password: env.GetString("AUTH_BASIC_PASSWORD", ""),
			},
		},
	}

	//Logger

	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()
	//Database

	db, err := db.New(cfg.db.dsn, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)

	defer db.Close()

	logger.Info("database connection pool established")
	if err != nil {
		logger.Fatal(err)
	}

	dbStore := store.NewPostgressStorage(db)
	mailer := mailer.NewMailTrapClient(
		cfg.mail.mailTrap.fromEmail,
		cfg.mail.mailTrap.smtpAddr,
		cfg.mail.mailTrap.smtpSandboxAddr,
		cfg.mail.mailTrap.username,
		cfg.mail.mailTrap.password,
		cfg.mail.mailTrap.smtpPort,
		logger,
	)

	tokenMaker, err := store.NewTokenStore(cfg.auth.AccessSecretKey, cfg.auth.RefreshSecretKey)

	if err != nil {
		logger.Panicf("setting up token maker error:  %w", err)
	}

	app := &application{
		config:     cfg,
		store:      dbStore,
		logger:     logger,
		mailer:     mailer,
		tokenMaker: tokenMaker,
	}

	mux := app.mount()
	log.Fatal(app.serve(mux))
}
