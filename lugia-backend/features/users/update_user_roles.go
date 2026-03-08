// Feature doc: docs/features/user-management.md, docs/features/audit-logging.md
package users

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"dislyze/jirachi/auditlog"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/authz"
	"lugia/lib/iputils"
	"lugia/lib/middleware"
	"lugia/queries"
)

var UpdateUserRolesOp = huma.Operation{
	OperationID: "update-user-roles",
	Method:      http.MethodPost,
	Path:        "/users/{userID}/roles",
}

type UpdateUserRolesInput struct {
	UserID string `path:"userID"`
	Body   UpdateUserRolesRequestBody
}

type UpdateUserRolesRequestBody struct {
	RoleIDs []string `json:"role_ids" minItems:"1"`
}

func (r *UpdateUserRolesRequestBody) Resolve(ctx huma.Context) []error {
	if r.RoleIDs == nil {
		return []error{fmt.Errorf("role_ids is required")}
	}
	return nil
}

func parseUUIDs(ids []string) ([]pgtype.UUID, error) {
	result := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		if err := result[i].Scan(id); err != nil {
			return nil, fmt.Errorf("invalid UUID %q: %w", id, err)
		}
	}
	return result, nil
}

// difference returns elements that are in slice1 but not in slice2
func difference(slice1, slice2 []pgtype.UUID) []pgtype.UUID {
	set := make(map[string]struct{})
	for _, item := range slice2 {
		set[item.String()] = struct{}{}
	}

	var result []pgtype.UUID
	for _, item := range slice1 {
		if _, exists := set[item.String()]; !exists {
			result = append(result, item)
		}
	}
	return result
}

func (h *UsersHandler) UpdateUserRoles(ctx context.Context, input *UpdateUserRolesInput) (*struct{}, error) {
	var targetUserID pgtype.UUID
	if err := targetUserID.Scan(input.UserID); err != nil {
		return nil, errlib.NewError(fmt.Errorf("invalid user ID format for update roles: %w", err), http.StatusBadRequest)
	}

	roleIDs, err := parseUUIDs(input.Body.RoleIDs)
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("invalid role ID format: %w", err), http.StatusBadRequest)
	}

	if err := h.updateUserRoles(ctx, targetUserID, roleIDs); err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *UsersHandler) updateUserRoles(ctx context.Context, targetUserID pgtype.UUID, roleIDs []pgtype.UUID) error {
	requestingUserID := libctx.GetUserID(ctx)
	requestingTenantID := libctx.GetTenantID(ctx)

	if requestingUserID == targetUserID {
		return errlib.NewError(fmt.Errorf("UpdateUserRoles: user %s attempting to update their own role", requestingUserID.String()), http.StatusBadRequest)
	}

	targetUser, err := h.q.GetUserByID(ctx, targetUserID)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.NewError(fmt.Errorf("UpdateUserRoles: target user with ID %s not found: %w", targetUserID.String(), err), http.StatusNotFound)
		}
		return errlib.NewError(fmt.Errorf("UpdateUserRoles: failed to get target user %s: %w", targetUserID.String(), err), http.StatusInternalServerError)
	}

	if requestingTenantID != targetUser.TenantID {
		return errlib.NewError(fmt.Errorf("UpdateUserRoles: requesting user %s (tenant %s) attempting to update user %s (tenant %s) in different tenant", requestingUserID.String(), requestingTenantID.String(), targetUserID.String(), targetUser.TenantID.String()), http.StatusForbidden)
	}

	validRoleIDs, err := h.q.ValidateRolesBelongToTenant(ctx, &queries.ValidateRolesBelongToTenantParams{
		Column1:  roleIDs,
		TenantID: requestingTenantID,
	})
	if err != nil {
		return errlib.NewError(fmt.Errorf("UpdateUserRoles: failed to validate roles belong to tenant: %w", err), http.StatusInternalServerError)
	}
	if len(validRoleIDs) != len(roleIDs) {
		return errlib.NewError(fmt.Errorf("UpdateUserRoles: some roles don't belong to tenant"), http.StatusBadRequest)
	}

	currentRoleIDs, err := h.q.GetUserRoleIDs(ctx, &queries.GetUserRoleIDsParams{
		UserID:   targetUserID,
		TenantID: requestingTenantID,
	})
	if err != nil {
		return errlib.NewError(fmt.Errorf("UpdateUserRoles: failed to get current user roles: %w", err), http.StatusInternalServerError)
	}

	toAdd := difference(roleIDs, currentRoleIDs)
	toRemove := difference(currentRoleIDs, roleIDs)

	if len(toRemove) > 0 || len(toAdd) > 0 {
		tx, err := h.dbConn.Begin(ctx)
		if err != nil {
			return errlib.NewError(fmt.Errorf("UpdateUserRoles: failed to begin transaction: %w", err), http.StatusInternalServerError)
		}
		defer func() {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
				errlib.LogError(fmt.Errorf("UpdateUserRoles: failed to rollback transaction: %w", rbErr))
			}
		}()

		qtx := h.q.WithTx(tx)

		if len(toRemove) > 0 {
			err = qtx.RemoveRolesFromUser(ctx, &queries.RemoveRolesFromUserParams{
				UserID:   targetUserID,
				TenantID: requestingTenantID,
				Column3:  toRemove,
			})
			if err != nil {
				return errlib.NewError(fmt.Errorf("UpdateUserRoles: failed to remove roles: %w", err), http.StatusInternalServerError)
			}
		}

		if len(toAdd) > 0 {
			addRolesInput := make([]*queries.AddRolesToUserParams, len(toAdd))
			for i, roleID := range toAdd {
				addRolesInput[i] = &queries.AddRolesToUserParams{
					UserID:   targetUserID,
					RoleID:   roleID,
					TenantID: requestingTenantID,
				}
			}

			_, err = qtx.AddRolesToUser(ctx, addRolesInput)
			if err != nil {
				return errlib.NewError(fmt.Errorf("UpdateUserRoles: failed to add roles: %w", err), http.StatusInternalServerError)
			}
		}

		if authz.TenantHasFeature(ctx, authz.FeatureAuditLog) {
			actorDBUser, err := qtx.GetUserByID(ctx, requestingUserID)
			if err != nil {
				return errlib.NewError(fmt.Errorf("UpdateUserRoles: failed to get actor user details for audit log: %w", err), http.StatusInternalServerError)
			}

			r := middleware.GetHTTPRequest(ctx)
			metadata, _ := json.Marshal(map[string]string{
				"actor_name":        actorDBUser.Name,
				"actor_email":       actorDBUser.Email,
				"target_user_name":  targetUser.Name,
				"target_user_email": targetUser.Email,
			})

			ipAddr, _ := netip.ParseAddr(iputils.ExtractClientIP(r))
			err = qtx.InsertAuditLog(ctx, &queries.InsertAuditLogParams{
				TenantID:     requestingTenantID,
				ActorID:      requestingUserID,
				ResourceType: string(auditlog.ResourceUser),
				Action:       string(auditlog.ActionRolesUpdated),
				Outcome:      string(auditlog.OutcomeSuccess),
				ResourceID:   pgtype.Text{String: targetUserID.String(), Valid: true},
				Metadata:     metadata,
				IpAddress:    &ipAddr,
				UserAgent:    pgtype.Text{String: r.UserAgent(), Valid: true},
			})
			if err != nil {
				return errlib.NewError(fmt.Errorf("UpdateUserRoles: failed to insert audit log: %w", err), http.StatusInternalServerError)
			}
		}

		if err = tx.Commit(ctx); err != nil {
			return errlib.NewError(fmt.Errorf("UpdateUserRoles: failed to commit transaction: %w", err), http.StatusInternalServerError)
		}
	}

	return nil
}
