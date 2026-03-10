package store

import (
	"context"
	"database/sql"
	"fmt"

	"go-sigil/internal/constants"
	"go-sigil/internal/models"
)

// ReplaceFileEdges atomically replaces all call edges where the caller symbol
// belongs to the given file. Deletes existing edges then inserts the new set
// in a single transaction.
func (s *SQLiteStore) ReplaceFileEdges(ctx context.Context, file string, edges []models.CallEdge) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		// Delete edges whose caller_id is a symbol in this file
		const delQ = `
DELETE FROM call_edges
WHERE caller_id IN (SELECT id FROM symbols WHERE file = ?)`
		if _, err := tx.ExecContext(ctx, delQ, file); err != nil {
			return fmt.Errorf("delete edges for file %s: %w", file, err)
		}

		if len(edges) == 0 {
			return nil
		}

		const insQ = `
INSERT OR IGNORE INTO call_edges (caller_id, callee_id, raw_expression, confidence)
VALUES (?, ?, ?, ?)`

		stmt, err := tx.PrepareContext(ctx, insQ)
		if err != nil {
			return fmt.Errorf("prepare edge insert: %w", err)
		}
		defer stmt.Close()

		for _, e := range edges {
			conf := e.Confidence
			if conf == "" {
				conf = string(constants.ConfidenceStatic)
			}
			if _, err := stmt.ExecContext(ctx,
				e.CallerID,
				nullableString(e.CalleeID),
				nullableString(e.RawExpression),
				conf,
			); err != nil {
				return fmt.Errorf("insert edge %s→%s: %w", e.CallerID, e.CalleeID, err)
			}
		}
		return nil
	})
}

// GetCalls returns edges where symbolID is the caller.
// depth > 1 multi-hop traversal is handled by the service layer.
func (s *SQLiteStore) GetCalls(ctx context.Context, symbolID string, _ int) ([]models.CallEdge, error) {
	const q = `SELECT caller_id, callee_id, raw_expression, confidence
               FROM call_edges WHERE caller_id = ?`
	rows, err := s.db.QueryContext(ctx, q, symbolID)
	if err != nil {
		return nil, fmt.Errorf("get calls for %s: %w", symbolID, err)
	}
	defer rows.Close()
	return scanEdges(rows)
}

// GetCalledBy returns edges where symbolID is the callee.
func (s *SQLiteStore) GetCalledBy(ctx context.Context, symbolID string, _ int) ([]models.CallEdge, error) {
	const q = `SELECT caller_id, callee_id, raw_expression, confidence
               FROM call_edges WHERE callee_id = ?`
	rows, err := s.db.QueryContext(ctx, q, symbolID)
	if err != nil {
		return nil, fmt.Errorf("get called_by for %s: %w", symbolID, err)
	}
	defer rows.Close()
	return scanEdges(rows)
}

func scanEdges(rows *sql.Rows) ([]models.CallEdge, error) {
	var edges []models.CallEdge
	for rows.Next() {
		var e models.CallEdge
		var calleeID, rawExpr sql.NullString
		if err := rows.Scan(&e.CallerID, &calleeID, &rawExpr, &e.Confidence); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		e.CalleeID = scanNullableString(calleeID)
		e.RawExpression = scanNullableString(rawExpr)
		edges = append(edges, e)
	}
	return edges, rows.Err()
}
