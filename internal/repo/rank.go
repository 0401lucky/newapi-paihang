package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// UserHit 个人查询的候选用户。
type UserHit struct {
	ID   int64  `json:"user_id"`
	Name string `json:"name"`
}

// SearchUsers 用 keyword LIKE 匹配 display_name / username，过滤隐藏。
func SearchUsers(ctx context.Context, db *sql.DB, keyword string, hidden []int64, limit int) ([]UserHit, error) {
	if keyword == "" {
		return nil, nil
	}
	hiddenClause, hiddenArgs := BuildHiddenClause(hidden)
	q := fmt.Sprintf(`
SELECT u.id, %s AS name
FROM users u
WHERE u.status=1 AND u.deleted_at IS NULL
  AND (u.display_name LIKE ? OR u.username LIKE ?)%s
ORDER BY u.id ASC
LIMIT ?`, DisplayNameSQL, hiddenClause)
	args := []any{"%" + keyword + "%", "%" + keyword + "%"}
	args = append(args, hiddenArgs...)
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()
	var out []UserHit
	for rows.Next() {
		var u UserHit
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// RichRankOf 该用户在土豪榜的 1-based 排名；未上榜返回 0。
func RichRankOf(ctx context.Context, db *sql.DB, userID int64, hidden []int64) (int, error) {
	return rankByUserField(ctx, db, userID, hidden, "quota")
}

// SpenderRankOf 按 used_quota 累计排名。
func SpenderRankOf(ctx context.Context, db *sql.DB, userID int64, hidden []int64) (int, error) {
	return rankByUserField(ctx, db, userID, hidden, "used_quota")
}

func rankByUserField(ctx context.Context, db *sql.DB, userID int64, hidden []int64, field string) (int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(hidden)
	var myVal int64
	row := db.QueryRowContext(ctx,
		"SELECT "+field+" FROM users WHERE id=? AND status=1 AND deleted_at IS NULL",
		userID)
	if err := row.Scan(&myVal); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	for _, h := range hidden {
		if h == userID {
			return 0, nil
		}
	}
	q := fmt.Sprintf(`
SELECT COUNT(*) + 1 FROM users u
WHERE u.status=1 AND u.deleted_at IS NULL
  AND u.%s > ?%s`, field, hiddenClause)
	args := []any{myVal}
	args = append(args, hiddenArgs...)
	var rank int
	if err := db.QueryRowContext(ctx, q, args...).Scan(&rank); err != nil {
		return 0, err
	}
	return rank, nil
}

// LogAggRankOf 通用于 logs 聚合的榜单（foodie/tokens/gourmet/biteking/nightowl）。
// metric: "count"|"distinct_model"|"max_quota"|"sum_tokens"|"night_count"
func LogAggRankOf(ctx context.Context, db *sql.DB, userID int64, r Range, metric string, hidden []int64) (int, error) {
	myVal, ok, err := logAggValue(ctx, db, userID, r, metric)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	for _, h := range hidden {
		if h == userID {
			return 0, nil
		}
	}
	hiddenClause, hiddenArgs := BuildHiddenClause(hidden)
	timeClause := ""
	timeArgs := []any{}
	if r != RangeAll {
		timeClause = " AND l.created_at >= ?"
		timeArgs = []any{RangeStart(r)}
	}
	aggExpr, extraWhere := metricExpr(metric)
	q := fmt.Sprintf(`
SELECT COUNT(*) + 1 FROM (
  SELECT %s AS v FROM logs l JOIN users u ON l.user_id=u.id
  WHERE l.type=2%s%s
    AND u.status=1 AND u.deleted_at IS NULL%s
  GROUP BY l.user_id
  HAVING v > ?
) t`, aggExpr, timeClause, extraWhere, hiddenClause)
	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, myVal)
	var rank int
	if err := db.QueryRowContext(ctx, q, args...).Scan(&rank); err != nil {
		return 0, err
	}
	return rank, nil
}

func metricExpr(metric string) (aggExpr, extraWhere string) {
	switch metric {
	case "count":
		return "COUNT(*)", ""
	case "distinct_model":
		return "COUNT(DISTINCT l.model_name)", ""
	case "max_quota":
		return "MAX(l.quota)", ""
	case "sum_tokens":
		return "SUM(l.prompt_tokens + l.completion_tokens)", ""
	case "night_count":
		return "COUNT(*)", " AND HOUR(CONVERT_TZ(FROM_UNIXTIME(l.created_at), '+00:00', '+08:00')) BETWEEN 0 AND 5"
	}
	return "COUNT(*)", ""
}

func logAggValue(ctx context.Context, db *sql.DB, userID int64, r Range, metric string) (float64, bool, error) {
	timeClause := ""
	timeArgs := []any{}
	if r != RangeAll {
		timeClause = " AND created_at >= ?"
		timeArgs = []any{RangeStart(r)}
	}
	aggExpr, extraWhere := metricExpr(metric)
	// metricExpr 里的 extraWhere 用了 l. 前缀但这里没 JOIN users，需要去掉 l. 前缀的逻辑：
	// 简化：在这里 logs 表别名也用 l 即可
	q := fmt.Sprintf("SELECT %s FROM logs l WHERE l.type=2 AND l.user_id=?%s%s",
		aggExpr, timeClause, extraWhere)
	args := append([]any{userID}, timeArgs...)
	var v sql.NullFloat64
	if err := db.QueryRowContext(ctx, q, args...).Scan(&v); err != nil {
		return 0, false, err
	}
	if !v.Valid || v.Float64 == 0 {
		return 0, false, nil
	}
	return v.Float64, true, nil
}
