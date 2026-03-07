// Generates the OpenAPI spec from huma endpoint definitions.
// Run: go run ./cmd/openapi
// Output: openapi.json in the lugia-backend directory.
//
// IMPORTANT: This file mirrors the route registrations in lugia-backend/main.go.
// When adding or removing a huma endpoint there, update this file too.
// The handlers here are no-ops — only the types matter for spec generation.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"lugia/features/auth"
	"lugia/features/ip_whitelist"
	"lugia/features/roles"
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

		// /auth endpoints
		huma.Register(api, auth.SignupOp, func(_ context.Context, _ *auth.SignupInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, auth.LoginOp, func(_ context.Context, _ *auth.LoginInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, auth.LogoutOp, func(_ context.Context, _ *auth.LogoutInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, auth.AcceptInviteOp, func(_ context.Context, _ *auth.AcceptInviteInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, auth.TenantSignupOp, func(_ context.Context, _ *auth.TenantSignupInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, auth.ForgotPasswordOp, func(_ context.Context, _ *auth.ForgotPasswordInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, auth.VerifyResetTokenOp, func(_ context.Context, _ *auth.VerifyResetTokenInput) (*auth.VerifyResetTokenOutput, error) {
			return nil, nil
		})
		huma.Register(api, auth.ResetPasswordOp, func(_ context.Context, _ *auth.ResetPasswordInput) (*struct{}, error) {
			return nil, nil
		})

		// /me endpoints
		huma.Register(api, users.GetMeOp, func(_ context.Context, _ *users.GetMeInput) (*users.GetMeOutput, error) {
			return nil, nil
		})
		huma.Register(api, users.UpdateMeOp, func(_ context.Context, _ *users.UpdateMeInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, users.ChangePasswordOp, func(_ context.Context, _ *users.ChangePasswordInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, users.ChangeEmailOp, func(_ context.Context, _ *users.ChangeEmailInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, users.VerifyChangeEmailOp, func(_ context.Context, _ *users.VerifyChangeEmailInput) (*struct{}, error) {
			return nil, nil
		})

		// /tenant endpoints
		huma.Register(api, users.ChangeTenantNameOp, func(_ context.Context, _ *users.ChangeTenantNameInput) (*struct{}, error) {
			return nil, nil
		})

		// /users endpoints
		huma.Register(api, users.GetUsersOp, func(_ context.Context, _ *users.GetUsersInput) (*users.GetUsersOutput, error) {
			return nil, nil
		})
		huma.Register(api, roles.GetUsersRolesOp, func(_ context.Context, _ *roles.GetRolesInput) (*roles.GetRolesOutput, error) {
			return nil, nil
		})
		huma.Register(api, users.InviteUserOp, func(_ context.Context, _ *users.InviteUserInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, users.ResendInviteOp, func(_ context.Context, _ *users.ResendInviteInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, users.UpdateUserRolesOp, func(_ context.Context, _ *users.UpdateUserRolesInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, users.DeleteUserOp, func(_ context.Context, _ *users.DeleteUserInput) (*struct{}, error) {
			return nil, nil
		})

		// /roles endpoints
		huma.Register(api, roles.GetRolesOp, func(_ context.Context, _ *roles.GetRolesInput) (*roles.GetRolesOutput, error) {
			return nil, nil
		})
		huma.Register(api, roles.GetPermissionsOp, func(_ context.Context, _ *roles.GetPermissionsInput) (*roles.GetPermissionsOutput, error) {
			return nil, nil
		})
		huma.Register(api, roles.CreateRoleOp, func(_ context.Context, _ *roles.CreateRoleInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, roles.UpdateRoleOp, func(_ context.Context, _ *roles.UpdateRoleInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, roles.DeleteRoleOp, func(_ context.Context, _ *roles.DeleteRoleInput) (*struct{}, error) {
			return nil, nil
		})

		// /ip-whitelist endpoints
		huma.Register(api, ip_whitelist.GetIPWhitelistOp, func(_ context.Context, _ *ip_whitelist.GetIPWhitelistInput) (*ip_whitelist.GetIPWhitelistOutput, error) {
			return nil, nil
		})
		huma.Register(api, ip_whitelist.AddIPOp, func(_ context.Context, _ *ip_whitelist.AddIPInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, ip_whitelist.UpdateIPLabelOp, func(_ context.Context, _ *ip_whitelist.UpdateIPLabelInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, ip_whitelist.DeleteIPOp, func(_ context.Context, _ *ip_whitelist.DeleteIPInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, ip_whitelist.ActivateWhitelistOp, func(_ context.Context, _ *ip_whitelist.ActivateWhitelistInput) (*ip_whitelist.ActivateWhitelistOutput, error) {
			return nil, nil
		})
		huma.Register(api, ip_whitelist.DeactivateWhitelistOp, func(_ context.Context, _ *ip_whitelist.DeactivateWhitelistInput) (*struct{}, error) {
			return nil, nil
		})
		huma.Register(api, ip_whitelist.EmergencyDeactivateOp, func(_ context.Context, _ *ip_whitelist.EmergencyDeactivateInput) (*struct{}, error) {
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
