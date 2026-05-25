package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// GourmetTop 美食家：调用过的不同模型种类最多。
func GourmetTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)
	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND l.created_at >= ?"
		timeArgs = []any{RangeStart(q.Range)}
	}
	countSQL := fmt.Sprintf(`
SELECT COUNT(DISTINCT l.user_id)
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s
  AND u.status=1 AND u.deleted_at IS NULL%s`, timeClause, hiddenClause)
	countArgs := append(timeArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gourmet count: %w", err)
	}
	listSQL := fmt.Sprintf(`
SELECT l.user_id, %s AS name, COUNT(DISTINCT l.model_name) AS variety
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s
  AND u.status=1 AND u.deleted_at IS NULL%s
GROUP BY l.user_id, name
ORDER BY variety DESC, l.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, timeClause, hiddenClause)
	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())
	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("gourmet query: %w", err)
	}
	defer rows.Close()
	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var variety int64
		if err := rows.Scan(&it.UserID, &it.Name, &variety); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(variety)
		it.ValueDisplay = fmt.Sprintf("%d 种", variety)
		items = append(items, it)
		rank++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
