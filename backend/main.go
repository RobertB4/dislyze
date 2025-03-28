package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"lugia/config"
	"lugia/db"
	"lugia/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func SetupRoutes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:1338"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", handlers.Signup)
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
	dbConfig := db.NewConfig(env)
	pool, err := db.NewDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.CloseDB(pool)

	// Create a channel to listen for errors coming from the server
	serverErrors := make(chan error, 1)

	// Create a channel to listen for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Setup routes
	router := SetupRoutes()

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
