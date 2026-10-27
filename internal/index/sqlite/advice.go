package sqliteindex

import (
	"context"
	"errors"
	"fmt"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/localintelligence"
)

// AdviceExcerpts returns bounded, ordered documentation candidates from a fresh
// derived index. Application policy performs the final path and privacy filter.
func AdviceExcerpts(ctx context.Context, root string) ([]localintelligence.Excerpt, error) {
	status, err := Inspect(ctx, root)
	if err != nil {
		return nil, err
	}
	if !status.Exists {
		return nil, errors.New("context index does not exist; run context rebuild")
	}
	if !status.Fresh {
		return nil, errors.New("context index is stale; run context rebuild")
	}
	_, databasePath, err := validatedPaths(root)
	if err != nil {
		return nil, err
	}
	db, err := openDatabase(databasePath, true)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT path, sha256, substr(content, 1, 8193)
		FROM artifacts
		WHERE content IS NOT NULL
		  AND (
		    path = 'README.md'
		    OR path = 'AGENTS.md'
		    OR (path LIKE 'docs/%' AND lower(path) LIKE '%.md')
		    OR (path LIKE '.agents/conventions/%' AND lower(path) LIKE '%.md')
		  )
		ORDER BY path
		LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("query adviser context: %w", err)
	}
	defer rows.Close()

	excerpts := make([]localintelligence.Excerpt, 0)
	for rows.Next() {
		var excerpt localintelligence.Excerpt
		if err := rows.Scan(&excerpt.Path, &excerpt.SHA256, &excerpt.Text); err != nil {
			return nil, fmt.Errorf("read adviser context: %w", err)
		}
		excerpts = append(excerpts, excerpt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate adviser context: %w", err)
	}
	return excerpts, nil
}
