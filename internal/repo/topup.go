package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// TopupTop 充值榜：top_ups SUM(money) WHERE status='success'。
func TopupTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)

	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND t.create_time >= ?"
		timeArgs = []any{RangeStart(q.Range)}
	}

	countSQL := fmt.Sprintf(`
SELECT COUNT(DISTINCT t.user_id)
FROM top_ups t JOIN users u ON t.user_id=u.id
WHERE t.status='success'%s
  AND u.status=1 AND u.deleted_at IS NULL%s`, timeClause, hiddenClause)
	countArgs := append(timeArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("topup count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT t.user_id, %s AS name, SUM(t.money) AS total_money
FROM top_ups t JOIN users u ON t.user_id=u.id
WHERE t.status='success'%s
  AND u.status=1 AND u.deleted_at IS NULL%s
GROUP BY t.user_id, name
ORDER BY total_money DESC, t.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, timeClause, hiddenClause)

	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("topup query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var money float64
		if err := rows.Scan(&it.UserID, &it.Name, &money); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = money
		it.ValueDisplay = FormatMoney(money)
		items = append(items, it)
		rank++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
