package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
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

func SetupRoutes(dbConn *pgxpool.Pool, env *config.Env) http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{env.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	authHandler := handlers.NewAuthHandler(dbConn, env)
	usersHandler := handlers.NewUsersHandler()

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", authHandler.Signup)
		r.Post("/login", authHandler.Login)
	})

	// Create rate limiter
	rateLimiter := ratelimit.NewRateLimiter(60*time.Minute, 5) // 5 attempts per 60 minutes

	// Create queries instance
	queries := queries.New(dbConn)

	// Protected routes
	r.Group(func(r chi.Router) {

		r.Use(middleware.NewAuthMiddleware(env, queries, rateLimiter, dbConn).Authenticate)

		// Users routes
		r.Route("/users", func(r chi.Router) {
			r.Get("/", usersHandler.GetUsers)
		})
	})

	return r
}

func main() {
	// Load environment variables
	env, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}

	// Initialize database connection
	pool, err := db.NewDB(env)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.CloseDB(pool)

	// Run migrations
	if err := db.RunMigrations(pool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create a channel to listen for errors coming from the server
	serverErrors := make(chan error, 1)

	// Create a channel to listen for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Setup routes with database connection and environment
	router := SetupRoutes(pool, env)

	// Start the server
	go func() {
		log.Printf("main: API listening on %s", ":1337")
		serverErrors <- http.ListenAndServe(":1337", router)
	}()

	// Blocking select waiting for either a server error or a signal
	select {
	case err := <-serverErrors:
		log.Fatalf("Error starting server: %v", err)

	case sig := <-sigChan:
		log.Printf("main: %v : Start shutdown", sig)
		os.Exit(0)
	}
}
