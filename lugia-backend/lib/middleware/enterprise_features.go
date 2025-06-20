package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	jirachiAuthz "dislyze/jirachi/authz"
	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"dislyze/jirachi/logger"
	"lugia/lib/authz"
	"lugia/queries"
)

func LoadEnterpriseFeatures(db *queries.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			tenantID := libctx.GetTenantID(ctx)
			userID := libctx.GetUserID(ctx)

			tenant, err := db.GetTenantByID(ctx, tenantID)
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
						Error:     "Tenant not found during tenant data loading",
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
					Error:     "Failed to load tenant data: " + err.Error(),
				})

				errlib.LogError(errlib.New(err, 500, "LoadTenantData: failed to get tenant by ID"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			enterpriseFeatures, err := ParseEnterpriseFeatures(tenant.EnterpriseFeatures)
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

				errlib.LogError(errlib.New(err, 500, "LoadTenantData: failed to parse enterprise features"))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Store enterprise features in context for downstream middlewares and handlers
			newCtx := libctx.WithEnterpriseFeatures(ctx, enterpriseFeatures)

			next.ServeHTTP(w, r.WithContext(newCtx))
		})
	}
}

func ParseEnterpriseFeatures(featuresJSON []byte) (*jirachiAuthz.EnterpriseFeatures, error) {
	if len(featuresJSON) == 0 {
		return &jirachiAuthz.EnterpriseFeatures{}, nil
	}

	var features jirachiAuthz.EnterpriseFeatures
	if err := json.Unmarshal(featuresJSON, &features); err != nil {
		return nil, fmt.Errorf("failed to unmarshal enterprise features: %w", err)
	}

	return &features, nil
}

func RequireFeature(feature authz.EnterpriseFeature) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authz.TenantHasFeature(r.Context(), feature) {
				userID := libctx.GetUserID(r.Context())
				tenantID := libctx.GetTenantID(r.Context())

				logger.LogAccessEvent(logger.AccessEvent{
					EventType: "feature",
					UserID:    userID.String(),
					TenantID:  tenantID.String(),
					IPAddress: r.RemoteAddr,
					UserAgent: r.UserAgent(),
					Timestamp: time.Now(),
					Success:   false,
					Error:     fmt.Sprintf("Feature not enabled: %s", feature),
					Feature:   string(feature),
				})

				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireRBAC() func(http.Handler) http.Handler {
	return RequireFeature(authz.FeatureRBAC)
}
