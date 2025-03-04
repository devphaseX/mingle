package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devphaseX/mingle.git/docs"
	"github.com/devphaseX/mingle.git/internal/mailer"
	"github.com/devphaseX/mingle.git/internal/ratelimiter"
	"github.com/devphaseX/mingle.git/internal/store"
	"github.com/devphaseX/mingle.git/internal/store/cache"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
)

type application struct {
	config       config
	store        store.Storage
	cacheStorage cache.Storage
	logger       *zap.SugaredLogger
	mailer       mailer.Client
	tokenMaker   store.TokenMaker
	rateLimiter  ratelimiter.RateLimiter
}

type config struct {
	addr        string
	db          dbConfig
	env         string
	apiURL      string
	frontendURL string
	mail        mailConfig
	auth        AuthConfig
	redisCfg    redisCfg
	rateLimiter ratelimiter.Config
}

type redisCfg struct {
	addr    string
	pw      string
	db      int
	enabled bool
}

type mailConfig struct {
	exp      time.Duration
	mailTrap mailTrapConfig
}

type mailTrapConfig struct {
	fromEmail       string
	smtpAddr        string
	smtpSandboxAddr string
	smtpPort        int
	apiKey          string
	username        string
	password        string
}

type dbConfig struct {
	dsn          string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type AuthConfig struct {
	AccessSecretKey  string
	RefreshSecretKey string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	RememberMeTTL    time.Duration
	basic            basicAuth
}

type basicAuth struct {
	username string
	password string
}

func (app *application) mount() *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(app.RateLimiterMiddleware)

	r.MethodNotAllowed(app.methodNotAllowedResponse)
	r.Route("/v1", func(r chi.Router) {
		r.With(app.BasicAuthMiddleware()).Get("/health", app.healthCheckHandler)

		docsURL := fmt.Sprintf("%s/swagger/doc.json", app.config.addr)
		r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL(docsURL)))

		r.Route("/posts", func(r chi.Router) {
			r.Use(app.AuthTokenMiddleware())
			r.Post("/", app.createPostHandler)

			r.Route("/{postID}", func(r chi.Router) {
				r.Use(app.postContextMiddleware)

				r.Get("/", app.getPostByIdHandler)
				r.Patch("/", app.checkPostOwnership("moderator", app.updatePostHandler))
				r.Delete("/", app.checkPostOwnership("admin", app.removePostByIdHandler))
			})
		})

		r.Route("/users", func(r chi.Router) {
			r.Put("/activate/{token}", app.activateUserHandler)
			r.Route("/{userID}", func(r chi.Router) {
				r.Use(app.AuthTokenMiddleware())
				r.Use(app.userContextMiddleware)

				r.Get("/", app.getUserByIdHandler)
				r.Put("/follow", app.followUserHandler)
				r.Put("/unfollow", app.unfollowUserHandler)
			})

			r.Group(func(r chi.Router) {
				r.Use(app.AuthTokenMiddleware())
				r.Get("/feed", app.getUserFeedHandler)
			})
		})

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", app.registerUserHandler)
			r.Post("/sign-in", app.signInHandler)
			r.Post("/refresh", app.refreshToken)
		})
	})

	return r
}

func (app *application) serve(mux http.Handler) error {
	docs.SwaggerInfo.Version = version
	docs.SwaggerInfo.Host = app.config.apiURL
	docs.SwaggerInfo.BasePath = "/v1"
	srv := &http.Server{
		Addr:         app.config.addr,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
		Handler:      mux,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		s := <-quit

		app.logger.Infow("caught signal", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)

		defer cancel()
		err := srv.Shutdown(ctx)

		if err != nil {
			shutdownError <- err
		}

		// app.logger.Infow("completing background tasks", "addr", srv.Addr)

	}()

	app.logger.Infow("server has started", "addr", app.config.addr, "env", app.config.env)

	err := srv.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	app.logger.Infow("server has stopped", "addr", app.config.addr, "env", app.config.env)

	return nil
}
