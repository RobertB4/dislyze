package middleware

import (
	jirachiAuthz "dislyze/jirachi/authz"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/logger"
	"encoding/json"
	"fmt"
	"lugia/queries"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// LoadContext loads shared request context data (enterprise features and user metadata)
// into the request context for use by downstream middlewares and handlers.
// This includes:
// - Tenant enterprise features (RBAC, IP whitelist, etc.)
// - User is_internal_user flag
func LoadTenantAndUserContext(db *queries.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			tenantID := libctx.GetTenantID(ctx)
			userID := libctx.GetUserID(ctx)

			contextData, err := db.GetTenantAndUserContext(ctx, &queries.GetTenantAndUserContextParams{
				TenantID: tenantID,
				UserID:   userID,
			})
			if err != nil {
				if errlib.Is(err, pgx.ErrNoRows) {
					logger.LogAccessEvent(logger.AccessEvent{
						EventType: "middleware",
						UserID:    userID.String(),
						TenantID:  tenantID.String(),
						IPAddress: r.RemoteAddr,
						UserAgent: r.UserAgent(),
						Timestamp: time.Now(),
						Success:   false,
						Error:     "Tenant or user not found during context loading",
					})

					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				logger.LogAccessEvent(logger.AccessEvent{
					EventType: "middleware",
					UserID:    userID.String(),
					TenantID:  tenantID.String(),
					IPAddress: r.RemoteAddr,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   false,
					Error:     "Failed to load context data: " + err.Error(),
				})

				errlib.LogError(errlib.New(err, 500, "LoadContext: failed to get context data"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			enterpriseFeatures, err := parseEnterpriseFeatures(contextData.EnterpriseFeatures)
			if err != nil {
				logger.LogAccessEvent(logger.AccessEvent{
					EventType: "middleware",
					UserID:    userID.String(),
					TenantID:  tenantID.String(),
					IPAddress: r.RemoteAddr,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   false,
					Error:     "Failed to parse enterprise features: " + err.Error(),
				})

				errlib.LogError(errlib.New(err, 500, "LoadContext: failed to parse enterprise features"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Store enterprise features and user metadata in context for downstream middlewares and handlers
			newCtx := libctx.WithEnterpriseFeatures(ctx, enterpriseFeatures)
			newCtx = libctx.WithIsInternalUser(newCtx, contextData.IsInternalUser)

			next.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}

func parseEnterpriseFeatures(featuresJSON []byte) (*jirachiAuthz.EnterpriseFeatures, error) {
	if len(featuresJSON) == 0 {
		return &jirachiAuthz.EnterpriseFeatures{}, nil
	}

	var features jirachiAuthz.EnterpriseFeatures
	if err := json.Unmarshal(featuresJSON, &features); err != nil {
		return nil, fmt.Errorf("failed to unmarshal enterprise features: %w", err)
	}

	return &features, nil
}
