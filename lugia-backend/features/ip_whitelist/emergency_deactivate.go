// Feature doc: docs/features/ip-whitelisting.md
package ip_whitelist

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"

	jirachiAuthz "dislyze/jirachi/authz"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/humautil"
	"lugia/lib/middleware"
	"lugia/queries"
)

var EmergencyDeactivateOp = huma.Operation{
	OperationID: "emergency-deactivate",
	Method:      http.MethodPost,
	Path:        "/ip-whitelist/emergency-deactivate",
}

type EmergencyDeactivateInput struct {
	Token string `query:"token"`
}

func (h *IPWhitelistHandler) EmergencyDeactivate(ctx context.Context, input *EmergencyDeactivateInput) (*struct{}, error) {
	r := middleware.GetHTTPRequest(ctx)

	if !h.rateLimiter.Allow(libctx.GetUserID(ctx).String(), r) {
		return nil, humautil.NewError(fmt.Errorf("rate limit exceeded for emergency deactivate"), http.StatusTooManyRequests)
	}

	if input.Token == "" {
		return nil, humautil.NewError(fmt.Errorf("emergency deactivate token is empty"), http.StatusBadRequest)
	}

	err := h.emergencyDeactivate(ctx, input.Token)
	if err != nil {
		var appErr *errlib.AppError
		if errlib.As(err, &appErr) {
			if appErr.Message != "" {
				return nil, humautil.NewErrorWithDetail(err, appErr.StatusCode, appErr.Message)
			}
			return nil, humautil.NewError(err, appErr.StatusCode)
		}
		return nil, humautil.NewError(err, http.StatusInternalServerError)
	}

	return nil, nil
}

func (h *IPWhitelistHandler) emergencyDeactivate(ctx context.Context, token string) error {
	claims, err := ValidateEmergencyToken(token, []byte(h.env.IPWhitelistEmergencyJWTSecret))
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: invalid emergency token: %w", err), http.StatusUnauthorized, "")
	}

	currentUserID := libctx.GetUserID(ctx)
	if currentUserID != claims.UserID {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: token user mismatch"), http.StatusForbidden, "")
	}

	tokenRecord, err := h.q.GetIPWhitelistEmergencyTokenByJTI(ctx, claims.JTI)
	if err != nil {
		if errlib.Is(err, pgx.ErrNoRows) {
			return errlib.New(fmt.Errorf("EmergencyDeactivate: emergency token not found"), http.StatusNotFound, "")
		}
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to get emergency token: %w", err), http.StatusInternalServerError, "")
	}

	if tokenRecord.UsedAt.Valid {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: emergency token already used"), http.StatusConflict, "")
	}

	tenant, err := h.q.GetTenantByID(ctx, claims.TenantID)
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to get tenant: %w", err), http.StatusInternalServerError, "")
	}

	if len(tenant.EnterpriseFeatures) == 0 {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: tenant has no enterprise features configured"), http.StatusInternalServerError, "")
	}

	var currentFeatures jirachiAuthz.EnterpriseFeatures
	if err := json.Unmarshal(tenant.EnterpriseFeatures, &currentFeatures); err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to parse enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	currentFeatures.IPWhitelist.Active = false

	updatedFeaturesJSON, err := json.Marshal(currentFeatures)
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to marshal enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	tx, err := h.dbConn.Begin(ctx)
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to begin transaction: %w", err), http.StatusInternalServerError, "")
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errlib.Is(rbErr, pgx.ErrTxClosed) && !errlib.Is(rbErr, sql.ErrTxDone) {
			errlib.LogError(fmt.Errorf("EmergencyDeactivate: failed to rollback transaction: %w", rbErr))
		}
	}()
	qtx := h.q.WithTx(tx)

	err = qtx.UpdateTenantEnterpriseFeatures(ctx, &queries.UpdateTenantEnterpriseFeaturesParams{
		EnterpriseFeatures: updatedFeaturesJSON,
		ID:                 claims.TenantID,
	})
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to update tenant enterprise features: %w", err), http.StatusInternalServerError, "")
	}

	err = qtx.MarkIPWhitelistEmergencyTokenAsUsed(ctx, claims.JTI)
	if err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to mark emergency token as used: %w", err), http.StatusInternalServerError, "")
	}

	if err := tx.Commit(ctx); err != nil {
		return errlib.New(fmt.Errorf("EmergencyDeactivate: failed to commit transaction: %w", err), http.StatusInternalServerError, "")
	}

	return nil
}

func ValidateEmergencyToken(tokenString string, secret []byte) (*EmergencyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &EmergencyClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid emergency token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("emergency token signature is invalid")
	}

	claims, ok := token.Claims.(*EmergencyClaims)
	if !ok {
		return nil, fmt.Errorf("emergency token claims are invalid")
	}

	if claims.Action != "ip_whitelist.emergency_deactivate" {
		return nil, fmt.Errorf("emergency token has invalid action: %s", claims.Action)
	}

	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf("emergency token has no ExpiresAt set")
	}

	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("emergency token has expired")
	}

	return claims, nil
}
