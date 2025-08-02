package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"lugia/features/auth"
	"lugia/features/ip_whitelist"
	"lugia/features/roles"
	"lugia/features/users"
	"lugia/lib/config"
	"lugia/lib/db"
	"lugia/lib/middleware"
	"lugia/queries"

	jirachi_auth "dislyze/jirachi/auth"
	"dislyze/jirachi/ratelimit"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// not used on localhost.
// for deployments, frontend gets embedded and built into the backend image
//
//go:embed frontend_embed
var frontendFiles embed.FS

func SetupRoutes(dbConn *pgxpool.Pool, env *config.Env, queries *queries.Queries) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	rateLimit, err := strconv.Atoi(env.AuthRateLimit)
	if err != nil {
		log.Fatalf("Failed to convert env.RateLimit to int: %v", err)
	}

	authRateLimiter := ratelimit.NewRateLimiter(5*time.Minute, rateLimit)
	resendInviteRateLimiter := ratelimit.NewRateLimiter(5*time.Minute, 1)
	deleteUserRateLimiter := ratelimit.NewRateLimiter(1*time.Minute, 10)
	changeEmailRateLimiter := ratelimit.NewRateLimiter(30*time.Minute, 1)
	ipWhitelistRateLimiter := ratelimit.NewRateLimiter(10*time.Minute, 30)

	authConfig := config.NewLugiaAuthConfig(env)
	jirachiAuthMiddleware := jirachi_auth.NewAuthMiddleware(authConfig, dbConn, authRateLimiter)

	authHandler := auth.NewAuthHandler(dbConn, env, authRateLimiter, queries)
	usersHandler := users.NewUsersHandler(dbConn, queries, env, resendInviteRateLimiter, deleteUserRateLimiter, changeEmailRateLimiter)
	rolesHandler := roles.NewRolesHandler(dbConn, queries, env)
	ipWhitelistHandler := ip_whitelist.NewIPWhitelistHandler(dbConn, queries, env, ipWhitelistRateLimiter)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Error writing health check response: %v", err)
		}
	})

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", authHandler.Signup)
			r.Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
			r.Post("/accept-invite", authHandler.AcceptInvite)
			r.Post("/tenant-signup", authHandler.TenantSignup)
			r.Post("/forgot-password", authHandler.ForgotPassword)
			r.Post("/verify-reset-token", authHandler.VerifyResetToken)
			r.Post("/reset-password", authHandler.ResetPassword)
		})

		r.Group(func(r chi.Router) {
			r.Use(jirachiAuthMiddleware.Authenticate)
			r.Use(middleware.LoadTenantAndUserContext(queries))
			r.Use(middleware.IPWhitelistMiddleware(queries))

			r.Get("/me", usersHandler.GetMe)
			r.Post("/me/change-name", usersHandler.UpdateMe)
			r.Post("/me/change-password", usersHandler.ChangePassword)
			r.Post("/me/change-email", usersHandler.ChangeEmail)
			r.Get("/me/verify-change-email", usersHandler.VerifyChangeEmail)

			r.Route("/users", func(r chi.Router) {
				r.With(middleware.RequireUsersView(queries)).Get("/", usersHandler.GetUsers)
				r.With(middleware.RequireUsersView(queries)).Get("/roles", rolesHandler.GetRoles)
				r.With(middleware.RequireUsersEdit(queries)).Post("/invite", usersHandler.InviteUser)
				r.With(middleware.RequireUsersEdit(queries)).Post("/{userID}/resend-invite", usersHandler.ResendInvite)
				r.With(middleware.RequireUsersEdit(queries)).Post("/{userID}/roles", usersHandler.UpdateUserRoles)
				r.With(middleware.RequireUsersEdit(queries)).Post("/{userID}/delete", usersHandler.DeleteUser)
			})

			r.Route("/roles", func(r chi.Router) {
				r.Use(middleware.RequireRBAC())

				r.With(middleware.RequireRolesView(queries)).Get("/", rolesHandler.GetRoles)
				r.With(middleware.RequireRolesView(queries)).Get("/permissions", rolesHandler.GetPermissions)
				r.With(middleware.RequireRolesEdit(queries)).Post("/create", rolesHandler.CreateRole)
				r.With(middleware.RequireRolesEdit(queries)).Post("/{roleID}/update", rolesHandler.UpdateRole)
				r.With(middleware.RequireRolesEdit(queries)).Post("/{roleID}/delete", rolesHandler.DeleteRole)
			})

			r.Route("/tenant", func(r chi.Router) {
				r.With(middleware.RequireTenantEdit(queries)).Post("/change-name", usersHandler.ChangeTenantName)
			})

			r.Route("/ip-whitelist", func(r chi.Router) {
				r.Use(middleware.RequireIPWhitelist())

				r.With(middleware.RequireIPWhitelistView(queries)).Get("/", ipWhitelistHandler.GetIPWhitelist)
				r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/create", ipWhitelistHandler.AddIPToWhitelist)
				r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/{id}/label/update", ipWhitelistHandler.UpdateIPLabel)
				r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/{id}/delete", ipWhitelistHandler.DeleteIP)
				r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/activate", ipWhitelistHandler.ActivateWhitelist)
				r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/deactivate", ipWhitelistHandler.DeactivateWhitelist)
				r.With(middleware.RequireIPWhitelistEdit(queries)).Post("/emergency-deactivate", ipWhitelistHandler.EmergencyDeactivate)
			})
		})
	})

	// Conditionally serve frontend static files - not used on localhost
	frontendFS, err := fs.Sub(frontendFiles, "frontend_embed")
	if err != nil {
		log.Printf("Failed to create frontend filesystem: %v", err)
		frontendFS = frontendFiles
	}

	// Check if frontend files exist
	if _, err := frontendFS.Open("app.html"); err == nil {
		log.Println("Frontend files found, enabling frontend routes")

		// Handle all non-API routes with frontend
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "app.html"
			}

			// Try to serve the requested file
			if file, err := frontendFS.Open(path); err == nil {
				defer file.Close()
				if stat, err := file.Stat(); err == nil && !stat.IsDir() {
					// File exists, serve it
					http.ServeFileFS(w, r, frontendFS, path)
					return
				}
			}

			// If file doesn't exist, serve app.html for SPA routing
			if indexFile, err := frontendFS.Open("app.html"); err == nil {
				defer indexFile.Close()
				if stat, err := indexFile.Stat(); err == nil && !stat.IsDir() {
					http.ServeFileFS(w, r, frontendFS, "app.html")
					return
				}
			}

			// Fallback if no frontend files
			w.WriteHeader(http.StatusNotFound)
		})
	} else {
		log.Println("No frontend files found, not serving frontend routes")
	}

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
