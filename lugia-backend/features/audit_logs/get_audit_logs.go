// Feature doc: docs/features/audit-logging.md
package audit_logs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5/pgtype"

	libctx "dislyze/jirachi/ctx"
	"dislyze/jirachi/errlib"
	"lugia/lib/conversions"
	"lugia/lib/pagination"
	"lugia/queries"
)

var GetAuditLogsOp = huma.Operation{
	OperationID: "get-audit-logs",
	Method:      http.MethodGet,
	Path:        "/audit-logs",
}

type AuditLogEntry struct {
	ID           string          `json:"id"`
	ActorID      string          `json:"actor_id"`
	ActorName    string          `json:"actor_name"`
	ActorEmail   string          `json:"actor_email"`
	ResourceType string          `json:"resource_type"`
	Action       string          `json:"action"`
	Outcome      string          `json:"outcome"`
	ResourceID   *string         `json:"resource_id"`
	Metadata     json.RawMessage `json:"metadata"`
	IPAddress    *string         `json:"ip_address"`
	UserAgent    *string         `json:"user_agent"`
	CreatedAt    string          `json:"created_at"`
}

type GetAuditLogsResponse struct {
	AuditLogs  []AuditLogEntry                `json:"audit_logs" nullable:"false"`
	Pagination pagination.PaginationMetadata   `json:"pagination"`
}

type GetAuditLogsInput struct {
	Page         int    `query:"page" default:"1" minimum:"1"`
	Limit        int    `query:"limit" default:"50" minimum:"1" maximum:"100"`
	ActorID      string `query:"actor_id"`
	ResourceType string `query:"resource_type"`
	Action       string `query:"action"`
	Outcome      string `query:"outcome"`
	FromDate     string `query:"from_date"`
	ToDate       string `query:"to_date"`
}

type GetAuditLogsOutput struct {
	Body GetAuditLogsResponse
}

func (h *AuditLogsHandler) GetAuditLogs(ctx context.Context, input *GetAuditLogsInput) (*GetAuditLogsOutput, error) {
	tenantID := libctx.GetTenantID(ctx)

	limit, err := conversions.SafeInt32(input.Limit)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid limit", err)
	}
	offset, err := conversions.SafeInt32((input.Page - 1) * input.Limit)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid page/limit combination", err)
	}

	filterParams, err := buildFilterParams(tenantID, input)
	if err != nil {
		return nil, err
	}

	totalCount, err := h.q.CountAuditLogs(ctx, &queries.CountAuditLogsParams{
		TenantID:     filterParams.TenantID,
		ActorID:      filterParams.ActorID,
		ResourceType: filterParams.ResourceType,
		Action:       filterParams.Action,
		Outcome:      filterParams.Outcome,
		FromDate:     filterParams.FromDate,
		ToDate:       filterParams.ToDate,
	})
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("GetAuditLogs: failed to count audit logs: %w", err), http.StatusInternalServerError)
	}

	rows, err := h.q.ListAuditLogs(ctx, &queries.ListAuditLogsParams{
		TenantID:     filterParams.TenantID,
		ActorID:      filterParams.ActorID,
		ResourceType: filterParams.ResourceType,
		Action:       filterParams.Action,
		Outcome:      filterParams.Outcome,
		FromDate:     filterParams.FromDate,
		ToDate:       filterParams.ToDate,
		LimitCount:   limit,
		OffsetCount:  offset,
	})
	if err != nil {
		return nil, errlib.NewError(fmt.Errorf("GetAuditLogs: failed to list audit logs: %w", err), http.StatusInternalServerError)
	}

	entries := make([]AuditLogEntry, len(rows))
	for i, row := range rows {
		entry := AuditLogEntry{
			ID:           row.ID.String(),
			ActorID:      row.ActorID.String(),
			ActorName:    row.ActorName,
			ActorEmail:   row.ActorEmail,
			ResourceType: row.ResourceType,
			Action:       row.Action,
			Outcome:      row.Outcome,
			Metadata:     row.Metadata,
			CreatedAt:    row.CreatedAt.Time.Format(time.RFC3339),
		}

		if row.ResourceID.Valid {
			entry.ResourceID = &row.ResourceID.String
		}
		if row.IpAddress != nil {
			s := row.IpAddress.String()
			entry.IPAddress = &s
		}
		if row.UserAgent.Valid {
			entry.UserAgent = &row.UserAgent.String
		}

		entries[i] = entry
	}

	paginationMetadata := pagination.CalculateMetadata(input.Page, limit, totalCount)

	return &GetAuditLogsOutput{
		Body: GetAuditLogsResponse{
			AuditLogs:  entries,
			Pagination: paginationMetadata,
		},
	}, nil
}

type filterParams struct {
	TenantID     pgtype.UUID
	ActorID      pgtype.UUID
	ResourceType string
	Action       string
	Outcome      string
	FromDate     pgtype.Timestamptz
	ToDate       pgtype.Timestamptz
}

func buildFilterParams(tenantID pgtype.UUID, input *GetAuditLogsInput) (*filterParams, error) {
	params := &filterParams{
		TenantID:     tenantID,
		ResourceType: input.ResourceType,
		Action:       input.Action,
		Outcome:      input.Outcome,
	}

	if input.ActorID != "" {
		actorUUID := pgtype.UUID{}
		if err := actorUUID.Scan(input.ActorID); err != nil {
			return nil, huma.Error400BadRequest("invalid actor_id format")
		}
		params.ActorID = actorUUID
	}

	if input.FromDate != "" {
		t, err := time.Parse(time.RFC3339, input.FromDate)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid from_date format, expected RFC3339")
		}
		params.FromDate = pgtype.Timestamptz{Time: t, Valid: true}
	}

	if input.ToDate != "" {
		t, err := time.Parse(time.RFC3339, input.ToDate)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid to_date format, expected RFC3339")
		}
		params.ToDate = pgtype.Timestamptz{Time: t, Valid: true}
	}

	return params, nil
}
