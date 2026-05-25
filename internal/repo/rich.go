package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// RichTop 返回钱包余额最多的用户。无时间维度（始终基于 users.quota 当前值）。
func RichTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)

	countSQL := "SELECT COUNT(*) FROM users u WHERE u.status=1 AND u.deleted_at IS NULL" + hiddenClause
	var total int
	if err := db.QueryRowContext(ctx, countSQL, hiddenArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("rich count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT u.id, %s AS name, u.quota
FROM users u
WHERE u.status=1 AND u.deleted_at IS NULL%s
ORDER BY u.quota DESC, u.id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, hiddenClause)

	args := append(hiddenArgs, q.PageSize, q.Offset())
	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("rich query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var quota int64
		if err := rows.Scan(&it.UserID, &it.Name, &quota); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(quota)
		it.ValueDisplay = FormatUSD(quota)
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}
