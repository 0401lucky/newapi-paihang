package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// FoodieTop 调用次数榜（logs COUNT(*) where type=2）。
func FoodieTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
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
		return nil, 0, fmt.Errorf("foodie count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT l.user_id, %s AS name, COUNT(*) AS cnt
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s
  AND u.status=1 AND u.deleted_at IS NULL%s
GROUP BY l.user_id, name
ORDER BY cnt DESC, l.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, timeClause, hiddenClause)

	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("foodie query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var cnt int64
		if err := rows.Scan(&it.UserID, &it.Name, &cnt); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(cnt)
		it.ValueDisplay = FormatCount(cnt) + " 次"
		items = append(items, it)
		rank++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
