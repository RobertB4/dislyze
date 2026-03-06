// Generates the OpenAPI spec from huma endpoint definitions.
// Run: go run ./cmd/openapi
// Output: openapi.json in the lugia-backend directory.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"lugia/features/users"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	config := huma.DefaultConfig("Lugia API", "1.0.0")

	// Mirror the /api route group from main.go so paths resolve correctly.
	r.Route("/api", func(r chi.Router) {
		api := humachi.New(r, config)

		// Register all huma endpoints with no-op handlers.
		// The handlers are never called — only the types matter for the spec.
		huma.Register(api, users.GetUsersOp, func(_ context.Context, _ *users.GetUsersInput) (*users.GetUsersOutput, error) {
			return nil, nil
		})
	})

	spec, err := json.MarshalIndent(config.OpenAPI, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile("openapi.json", spec, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write openapi.json: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("openapi.json generated")
}
