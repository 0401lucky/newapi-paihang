package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// LoyalTop 死忠粉：单模型调用占比 >= LoyalThreshold 且总调用 >= LoyalMinCalls。
func LoyalTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)
	since := RangeStart(q.Range)

	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND created_at >= ?"
		timeArgs = []any{since}
	}

	base := fmt.Sprintf(`
WITH per_model AS (
  SELECT user_id, model_name, COUNT(*) AS cnt
  FROM logs WHERE type=2%s
  GROUP BY user_id, model_name
),
ranked AS (
  SELECT user_id, model_name, cnt,
         SUM(cnt) OVER (PARTITION BY user_id) AS total_cnt,
         ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY cnt DESC) AS rn
  FROM per_model
),
agg AS (
  SELECT r.user_id, r.model_name AS top_model, r.cnt AS top_cnt, r.total_cnt,
         r.cnt * 1.0 / r.total_cnt AS ratio
  FROM ranked r WHERE r.rn = 1
)
`, timeClause)

	countSQL := base + fmt.Sprintf(`
SELECT COUNT(*) FROM agg
JOIN users u ON agg.user_id=u.id
WHERE agg.ratio >= ? AND agg.total_cnt >= ?
  AND u.status=1 AND u.deleted_at IS NULL%s`, hiddenClause)
	countArgs := append([]any{}, timeArgs...)
	countArgs = append(countArgs, q.LoyalThreshold, q.LoyalMinCalls)
	countArgs = append(countArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("loyal count: %w", err)
	}

	listSQL := base + fmt.Sprintf(`
SELECT agg.user_id, %s AS name, agg.ratio, agg.top_model, agg.total_cnt
FROM agg JOIN users u ON agg.user_id=u.id
WHERE agg.ratio >= ? AND agg.total_cnt >= ?
  AND u.status=1 AND u.deleted_at IS NULL%s
ORDER BY agg.ratio DESC, agg.total_cnt DESC, agg.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, hiddenClause)

	args := append([]any{}, timeArgs...)
	args = append(args, q.LoyalThreshold, q.LoyalMinCalls)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("loyal query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var ratio float64
		var topModel string
		var totalCnt int64
		if err := rows.Scan(&it.UserID, &it.Name, &ratio, &topModel, &totalCnt); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = ratio
		it.ValueDisplay = fmt.Sprintf("%.1f%%", ratio*100)
		it.Extra = map[string]any{"model": topModel, "total_calls": totalCnt}
		items = append(items, it)
		rank++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
