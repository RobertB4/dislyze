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
	"lugia/lib/humautil"
	"lugia/lib/middleware"
	"lugia/queries"

	jirachi_auth "dislyze/jirachi/auth"
	"dislyze/jirachi/ratelimit"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// not used on localhost.
// for deployments, frontend gets embedded and built into the backend image
//
//go:embed frontend_embed/*
var frontendFiles embed.FS

func SetupRoutes(dbConn *pgxpool.Pool, env *config.Env, queries *queries.Queries) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	rateLimit, err := strconv.Atoi(env.AuthRateLimit)
	if err != nil {
		log.Fatalf("Failed to convert env.RateLimit to int: %v", err)
	}

	authRateLimiter := ratelimit.NewRateLimiter("lugia", 5*time.Minute, rateLimit)
	resendInviteRateLimiter := ratelimit.NewRateLimiter("lugia", 5*time.Minute, 1)
	deleteUserRateLimiter := ratelimit.NewRateLimiter("lugia", 1*time.Minute, 10)
	changeEmailRateLimiter := ratelimit.NewRateLimiter("lugia", 30*time.Minute, 1)
	ipWhitelistRateLimiter := ratelimit.NewRateLimiter("lugia", 10*time.Minute, 30)

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
		// SSO auth endpoints (chi-style, not migrated to huma)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/sso/login", authHandler.SSOLogin)
			r.Post("/sso/acs", authHandler.SSOACS)
			r.Get("/sso/metadata", authHandler.SSOMetadata)
		})

		// Huma route registrations below are mirrored in lugia-backend/cmd/openapi/main.go
		// for OpenAPI spec generation. When adding or removing endpoints, update both files.
		humaConfig := humautil.NewConfig("Lugia API", "1.0.0")

		// /auth endpoints — public, only need InjectRawHTTP for cookie/rate-limit access
		authAPI := humachi.New(r.With(middleware.InjectRawHTTP), humaConfig)
		huma.Register(authAPI, auth.SignupOp, authHandler.Signup)
		huma.Register(authAPI, auth.LoginOp, authHandler.Login)
		huma.Register(authAPI, auth.LogoutOp, authHandler.Logout)
		huma.Register(authAPI, auth.AcceptInviteOp, authHandler.AcceptInvite)
		huma.Register(authAPI, auth.TenantSignupOp, authHandler.TenantSignup)
		huma.Register(authAPI, auth.ForgotPasswordOp, authHandler.ForgotPassword)
		huma.Register(authAPI, auth.VerifyResetTokenOp, authHandler.VerifyResetToken)
		huma.Register(authAPI, auth.ResetPasswordOp, authHandler.ResetPassword)

		// Authenticated huma endpoints — all registered at the /api level
		// to avoid chi sub-router path duplication.
		authenticatedMiddleware := chi.Chain(
			jirachiAuthMiddleware.Authenticate,
			middleware.LoadTenantAndUserContext(queries),
			middleware.IPWhitelistMiddleware(queries),
			middleware.InjectRawHTTP,
		)

		// /me endpoints — authenticated, no extra permission middleware
		meAPI := humachi.New(r.With(authenticatedMiddleware...), humaConfig)
		huma.Register(meAPI, users.GetMeOp, usersHandler.GetMe)
		huma.Register(meAPI, users.UpdateMeOp, usersHandler.UpdateMe)
		huma.Register(meAPI, users.ChangePasswordOp, usersHandler.ChangePassword)
		huma.Register(meAPI, users.ChangeEmailOp, usersHandler.ChangeEmail)
		huma.Register(meAPI, users.VerifyChangeEmailOp, usersHandler.VerifyChangeEmail)

		// /tenant endpoints
		tenantEditAPI := humachi.New(r.With(append(authenticatedMiddleware, middleware.RequireTenantEdit(queries))...), humaConfig)
		huma.Register(tenantEditAPI, users.ChangeTenantNameOp, usersHandler.ChangeTenantName)

		// /users endpoints
		usersViewAPI := humachi.New(r.With(append(authenticatedMiddleware, middleware.RequireUsersView(queries))...), humaConfig)
		huma.Register(usersViewAPI, users.GetUsersOp, usersHandler.GetUsers)
		huma.Register(usersViewAPI, roles.GetUsersRolesOp, rolesHandler.GetRoles)

		usersEditAPI := humachi.New(r.With(append(authenticatedMiddleware, middleware.RequireUsersEdit(queries))...), humaConfig)
		huma.Register(usersEditAPI, users.InviteUserOp, usersHandler.InviteUser)
		huma.Register(usersEditAPI, users.ResendInviteOp, usersHandler.ResendInvite)
		huma.Register(usersEditAPI, users.UpdateUserRolesOp, usersHandler.UpdateUserRoles)
		huma.Register(usersEditAPI, users.DeleteUserOp, usersHandler.DeleteUser)

		// /roles endpoints
		rolesViewAPI := humachi.New(r.With(append(authenticatedMiddleware, middleware.RequireRBAC(), middleware.RequireRolesView(queries))...), humaConfig)
		huma.Register(rolesViewAPI, roles.GetRolesOp, rolesHandler.GetRoles)
		huma.Register(rolesViewAPI, roles.GetPermissionsOp, rolesHandler.GetPermissions)

		rolesEditAPI := humachi.New(r.With(append(authenticatedMiddleware, middleware.RequireRBAC(), middleware.RequireRolesEdit(queries))...), humaConfig)
		huma.Register(rolesEditAPI, roles.CreateRoleOp, rolesHandler.CreateRole)
		huma.Register(rolesEditAPI, roles.UpdateRoleOp, rolesHandler.UpdateRole)
		huma.Register(rolesEditAPI, roles.DeleteRoleOp, rolesHandler.DeleteRole)

		// /ip-whitelist endpoints
		ipViewAPI := humachi.New(r.With(append(authenticatedMiddleware, middleware.RequireIPWhitelist(), middleware.RequireIPWhitelistView(queries))...), humaConfig)
		huma.Register(ipViewAPI, ip_whitelist.GetIPWhitelistOp, ipWhitelistHandler.GetIPWhitelist)

		ipEditAPI := humachi.New(r.With(append(authenticatedMiddleware, middleware.RequireIPWhitelist(), middleware.RequireIPWhitelistEdit(queries))...), humaConfig)
		huma.Register(ipEditAPI, ip_whitelist.AddIPOp, ipWhitelistHandler.AddIPToWhitelist)
		huma.Register(ipEditAPI, ip_whitelist.UpdateIPLabelOp, ipWhitelistHandler.UpdateIPLabel)
		huma.Register(ipEditAPI, ip_whitelist.DeleteIPOp, ipWhitelistHandler.DeleteIP)
		huma.Register(ipEditAPI, ip_whitelist.ActivateWhitelistOp, ipWhitelistHandler.ActivateWhitelist)
		huma.Register(ipEditAPI, ip_whitelist.DeactivateWhitelistOp, ipWhitelistHandler.DeactivateWhitelist)
		huma.Register(ipEditAPI, ip_whitelist.EmergencyDeactivateOp, ipWhitelistHandler.EmergencyDeactivate)
	})

	// Conditionally serve frontend static files - not used on localhost
	frontendFS, err := fs.Sub(frontendFiles, "frontend_embed")
	if err != nil {
		log.Printf("Failed to create frontend filesystem: %v", err)
		frontendFS = frontendFiles
	}

	// Check if frontend files exist (do not exist on localhost)
	if _, err := frontendFS.Open("app.html"); err == nil {
		log.Println("Frontend files found, serving frontend as SPA")

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/")

			// If file exists, serve it
			if file, err := frontendFS.Open(path); err == nil {
				if closeErr := file.Close(); closeErr != nil {
					log.Printf("Error closing file when trying to serve static file: %v", closeErr)
				}
				http.FileServer(http.FS(frontendFS)).ServeHTTP(w, r)
				return
			}

			// if file doesn't exist, return 404 for assets (e.g. .js, .css)
			if strings.Contains(path, ".") {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}

			// Fallback to app.html
			r.URL.Path = "/app.html"
			http.FileServer(http.FS(frontendFS)).ServeHTTP(w, r)
		})
	} else {
		log.Println("No frontend files found, not serving frontend as SPA")
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
