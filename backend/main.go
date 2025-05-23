package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"lugia/handlers"
	"lugia/lib/config"
	"lugia/lib/db"
	"lugia/lib/middleware"
	"lugia/lib/ratelimit"
	"lugia/queries"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRoutes(dbConn *pgxpool.Pool, env *config.Env, queries *queries.Queries) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{env.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	rateLimit, err := strconv.Atoi(env.RateLimit)
	if err != nil {
		log.Fatalf("Failed to convert env.RateLimit to int: %v", err)
	}

	rateLimiter := ratelimit.NewRateLimiter(60*time.Minute, rateLimit)

	authHandler := handlers.NewAuthHandler(dbConn, env, rateLimiter, queries)
	usersHandler := handlers.NewUsersHandler(dbConn, queries, env)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", authHandler.Signup)
		r.Post("/login", authHandler.Login)
		r.Post("/logout", authHandler.Logout)
		r.Post("/accept-invite", authHandler.AcceptInvite)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.NewAuthMiddleware(env, queries, rateLimiter, dbConn).Authenticate)

		r.Route("/users", func(r chi.Router) {
			r.Get("/", usersHandler.GetUsers)
			r.Post("/invite", usersHandler.InviteUser)
		})
	})

	return r
}

func main() {
	env, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}

	pool, err := db.NewDB(env)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.CloseDB(pool)

	if err := db.RunMigrations(pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	appQueries := queries.New(pool)

	serverErrors := make(chan error, 1)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	router := SetupRoutes(pool, env, appQueries)

	go func() {
		log.Printf("main: API listening on %s", ":1337")
		serverErrors <- http.ListenAndServe(":1337", router)
	}()

	select {
	case err := <-serverErrors:
		log.Fatalf("Error starting server: %v", err)

	case sig := <-sigChan:
		log.Printf("main: %v : Start shutdown", sig)
		os.Exit(0)
	}
}
