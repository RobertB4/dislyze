package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"lugia/features/auth"
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
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Error writing health check response: %v", err)
		}
	})

	rateLimit, err := strconv.Atoi(env.AuthRateLimit)
	if err != nil {
		log.Fatalf("Failed to convert env.RateLimit to int: %v", err)
	}

	authRateLimiter := ratelimit.NewRateLimiter(5*time.Minute, rateLimit)
	resendInviteRateLimiter := ratelimit.NewRateLimiter(5*time.Minute, 1)
	deleteUserRateLimiter := ratelimit.NewRateLimiter(1*time.Minute, 10)
	changeEmailRateLimiter := ratelimit.NewRateLimiter(30*time.Minute, 1)

	authHandler := auth.NewAuthHandler(dbConn, env, authRateLimiter, queries)
	usersHandler := handlers.NewUsersHandler(dbConn, queries, env, resendInviteRateLimiter, deleteUserRateLimiter, changeEmailRateLimiter)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", authHandler.Signup)
		r.Post("/login", authHandler.Login)
		r.Post("/logout", authHandler.Logout)
		r.Post("/accept-invite", authHandler.AcceptInvite)
		r.Post("/forgot-password", authHandler.ForgotPassword)
		r.Post("/verify-reset-token", authHandler.VerifyResetToken)
		r.Post("/reset-password", authHandler.ResetPassword)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.NewAuthMiddleware(env, queries, authRateLimiter, dbConn).Authenticate)

		r.Get("/me", usersHandler.GetMe)
		r.Post("/me/change-name", usersHandler.UpdateMe)
		r.Post("/me/change-password", usersHandler.ChangePassword)
		r.Post("/me/change-email", usersHandler.ChangeEmail)
		r.Get("/me/verify-change-email", usersHandler.VerifyChangeEmail)

		r.Route("/users", func(r chi.Router) {
			r.Use(middleware.RequireAdmin)

			r.Get("/", usersHandler.GetUsers)
			r.Post("/invite", usersHandler.InviteUser)
			r.Post("/{userID}/resend-invite", usersHandler.ResendInvite)
			r.Post("/{userID}/delete", usersHandler.DeleteUser)
			r.Post("/{userID}/permissions", usersHandler.UpdateUserPermissions)
		})

		r.Route("/tenant", func(r chi.Router) {
			r.Use(middleware.RequireAdmin)

			r.Post("/change-name", usersHandler.UpdateTenantName)
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
