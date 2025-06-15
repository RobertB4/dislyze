package main

import (
	"context"
	"fmt"
	"giratina/features/tenants"
	"giratina/features/users"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"giratina/features/auth"
	"giratina/lib/config"
	"giratina/lib/db"
	"giratina/queries"

	jirachi_auth "dislyze/jirachi/auth"
	"dislyze/jirachi/ratelimit"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRoutes(dbConn *pgxpool.Pool, env *config.Env, queries *queries.Queries) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	rateLimit, err := strconv.Atoi(env.AuthRateLimit)
	if err != nil {
		log.Fatalf("Failed to convert env.AuthRateLimit to int: %v", err)
	}

	authRateLimiter := ratelimit.NewRateLimiter(5*time.Minute, rateLimit)

	authConfig := config.NewGiratinaAuthConfig(env)
	jirachiAuthMiddleware := jirachi_auth.NewAuthMiddleware(authConfig, dbConn, authRateLimiter)
	authHandler := auth.NewAuthHandler(dbConn, env, authRateLimiter, queries)

	usersHandler := users.NewUsersHandler(dbConn, env, queries)
	tenantsHandler := tenants.NewTenantsHandler(dbConn, env, queries)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Error writing health check response: %v", err)
		}
	})

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
		})

		r.Group(func(r chi.Router) {
			r.Use(jirachiAuthMiddleware.Authenticate)

			r.Get("/me", usersHandler.GetMe)

			r.Route("/tenants", func(r chi.Router) {
				r.Get("/", tenantsHandler.GetTenants)
				r.Post("/{id}/update", tenantsHandler.UpdateTenant)
				r.Get("/{tenantID}/login", tenantsHandler.LogInToTenant)
				r.Post("/generate-token", tenantsHandler.GenerateTenantInvitationToken)
			})

			r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(`{"message": "protected users endpoint"}`)); err != nil {
					log.Printf("Error writing users response: %v", err)
				}
			})
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

	q := queries.New(pool)

	serverErrors := make(chan error, 1)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	router := SetupRoutes(pool, env, q)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", env.Port),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("main: API listening on %s", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}

	case sig := <-sigChan:
		log.Printf("main: %v : Start shutdown", sig)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("main: Graceful shutdown failed: %v", err)
		} else {
			log.Printf("main: Server gracefully stopped")
		}
	}
	log.Printf("main: Shutdown complete")
}
