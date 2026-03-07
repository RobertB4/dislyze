// Generates the OpenAPI spec from huma endpoint definitions.
// Run: go run ./cmd/openapi
// Output: openapi.json in the giratina-backend directory.
//
// IMPORTANT: This file mirrors the route registrations in giratina-backend/main.go.
// When adding or removing a huma endpoint there, update this file too.
// The handlers here are no-ops — only the types matter for spec generation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"giratina/features/auth"
	"giratina/features/tenants"
	"giratina/features/users"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	config := huma.DefaultConfig("Giratina API", "1.0.0")

	// Mirror the /api route group from main.go so paths resolve correctly.
	r.Route("/api", func(r chi.Router) {
		api := humachi.New(r, config)

		// Register all huma endpoints with no-op handlers.
		// The handlers are never called — only the types matter for the spec.

		// /auth endpoints
		huma.Register(api, auth.LoginOp, func(_ context.Context, _ *auth.LoginInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, auth.LogoutOp, func(_ context.Context, _ *auth.LogoutInput) (*struct{}, error) {
			return nil, nil
		})

		// /me endpoint
		huma.Register(api, users.GetMeOp, func(_ context.Context, _ *users.GetMeInput) (*users.GetMeOutput, error) {
			return nil, nil
		})

		// /tenants endpoints
		huma.Register(api, tenants.GetTenantsOp, func(_ context.Context, _ *tenants.GetTenantsInput) (*tenants.GetTenantsOutput, error) {
			return nil, nil
		})
		huma.Register(api, tenants.UpdateTenantOp, func(_ context.Context, _ *tenants.UpdateTenantInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, tenants.GenerateTokenOp, func(_ context.Context, _ *tenants.GenerateTokenInput) (*tenants.GenerateTokenOutput, error) {
			return nil, nil
		})
		huma.Register(api, tenants.GetUsersByTenantOp, func(_ context.Context, _ *tenants.GetUsersByTenantInput) (*tenants.GetUsersByTenantOutput, error) {
			return nil, nil
		})
		huma.Register(api, tenants.LogInToTenantOp, func(_ context.Context, _ *tenants.LogInToTenantInput) (*tenants.LogInToTenantOutput, error) {
			return nil, nil
		})
	})

	spec, err := json.MarshalIndent(config.OpenAPI, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile("openapi.json", spec, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write openapi.json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("openapi.json generated")
}
