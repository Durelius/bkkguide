// Package audit records every admin action.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
)

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Log writes one entry. Call it with the same transaction as the change it
// describes, so the change and its record commit together.
func Log(ctx context.Context, db execer, adminID, action, entity, entityID string, details any) error {
	if details == nil {
		details = map[string]any{}
	}
	b, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO audit_log (admin_id, action, entity, entity_id, details) VALUES (?, ?, ?, ?, ?)`,
		adminID, action, entity, entityID, string(b))
	return err
}

// Change is a before/after pair for one field in an update's details.
type Change struct {
	From any `json:"from"`
	To   any `json:"to"`
}
