package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// SpenderTop 散财榜：累计消费 (RangeAll 用 users.used_quota；其余走 logs SUM)。
func SpenderTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	if q.Range == RangeAll {
		return spenderAllTime(ctx, db, q)
	}
	return spenderWindowed(ctx, db, q)
}

func spenderAllTime(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)

	countSQL := "SELECT COUNT(*) FROM users u WHERE u.status=1 AND u.deleted_at IS NULL" + hiddenClause
	var total int
	if err := db.QueryRowContext(ctx, countSQL, hiddenArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("spender count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT u.id, %s AS name, u.used_quota
FROM users u
WHERE u.status=1 AND u.deleted_at IS NULL%s
ORDER BY u.used_quota DESC, u.id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, hiddenClause)

	args := append(hiddenArgs, q.PageSize, q.Offset())
	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("spender query: %w", err)
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
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func spenderWindowed(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)
	since := RangeStart(q.Range)

	countSQL := fmt.Sprintf(`
SELECT COUNT(DISTINCT l.user_id)
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2 AND l.created_at >= ?
  AND u.status=1 AND u.deleted_at IS NULL%s`, hiddenClause)
	countArgs := append([]any{since}, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("spender count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT l.user_id, %s AS name, SUM(l.quota) AS spent
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2 AND l.created_at >= ?
  AND u.status=1 AND u.deleted_at IS NULL%s
GROUP BY l.user_id, name
ORDER BY spent DESC, l.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, hiddenClause)
	args := append([]any{since}, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("spender query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var spent int64
		if err := rows.Scan(&it.UserID, &it.Name, &spent); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(spent)
		it.ValueDisplay = FormatUSD(spent)
		items = append(items, it)
		rank++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
