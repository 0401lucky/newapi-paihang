package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// BitekingTop 单笔王：每个用户的单次最大 quota 消耗。
func BitekingTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)
	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND l.created_at >= ?"
		timeArgs = []any{RangeStart(q.Range)}
	}
	base := fmt.Sprintf(`
WITH max_per_user AS (
  SELECT user_id, model_name, quota,
         ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY quota DESC, id ASC) AS rn
  FROM logs l
  WHERE l.type=2%s
)
`, timeClause)
	countSQL := base + fmt.Sprintf(`
SELECT COUNT(*) FROM max_per_user m
JOIN users u ON m.user_id=u.id
WHERE m.rn=1
  AND u.status=1 AND u.deleted_at IS NULL%s`, hiddenClause)
	countArgs := append([]any{}, timeArgs...)
	countArgs = append(countArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("biteking count: %w", err)
	}
	listSQL := base + fmt.Sprintf(`
SELECT m.user_id, %s AS name, m.quota, m.model_name
FROM max_per_user m JOIN users u ON m.user_id=u.id
WHERE m.rn=1
  AND u.status=1 AND u.deleted_at IS NULL%s
ORDER BY m.quota DESC, m.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, hiddenClause)
	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())
	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("biteking query: %w", err)
	}
	defer rows.Close()
	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var quota int64
		var model string
		if err := rows.Scan(&it.UserID, &it.Name, &quota, &model); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(quota)
		it.ValueDisplay = FormatUSD(quota)
		it.Extra = map[string]any{"model": model}
		items = append(items, it)
		rank++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
