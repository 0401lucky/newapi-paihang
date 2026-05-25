// Package repo 提供 9 个榜单 + 个人查询的数据访问层。
// 所有查询走只读账号，禁止 Exec/UPDATE/DELETE。
package repo

import (
	"strconv"
	"strings"
	"time"
)

// CSTZone 熬夜冠军和 today 计算硬编码 +08:00（北京时间）。
var CSTZone = time.FixedZone("CST", 8*3600)

// Range 时间窗类型
type Range string

const (
	RangeToday Range = "today"
	Range7d    Range = "7d"
	Range30d   Range = "30d"
	RangeAll   Range = "all"
)

// ValidRange 判断字符串是否为合法 Range。
func ValidRange(r string) bool {
	switch Range(r) {
	case RangeToday, Range7d, Range30d, RangeAll:
		return true
	}
	return false
}

// LeaderboardItem 是 9 个榜单的统一行结构。
type LeaderboardItem struct {
	Rank         int     `json:"rank"`
	UserID       int64   `json:"user_id"`
	Name         string  `json:"name"`
	Value        float64 `json:"value"`
	ValueDisplay string  `json:"value_display"`
	Extra        any     `json:"extra,omitempty"`
}

// QueryParams 所有榜单共用的查询参数。
type QueryParams struct {
	Range          Range
	Page           int
	PageSize       int
	HiddenUserIDs  []int64
	LoyalThreshold float64 // 仅死忠粉用
	LoyalMinCalls  int     // 仅死忠粉用
}

// Offset 分页偏移
func (q QueryParams) Offset() int { return (q.Page - 1) * q.PageSize }

// BuildHiddenClause 返回 (clause, args)，hidden 为空时返回空串。
// 调用方需把 args 拼到查询 args 末尾对应位置。
func BuildHiddenClause(hidden []int64) (string, []any) {
	if len(hidden) == 0 {
		return "", nil
	}
	placeholders := strings.Repeat("?,", len(hidden))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(hidden))
	for i, id := range hidden {
		args[i] = id
	}
	return " AND u.id NOT IN (" + placeholders + ")", args
}

// RangeStart 返回时间窗起始 Unix 秒；RangeAll 返回 0。
func RangeStart(r Range) int64 {
	return rangeStartAt(r, time.Now())
}

func rangeStartAt(r Range, now time.Time) int64 {
	switch r {
	case RangeToday:
		cst := now.In(CSTZone)
		t0 := time.Date(cst.Year(), cst.Month(), cst.Day(), 0, 0, 0, 0, CSTZone)
		return t0.Unix()
	case Range7d:
		return now.Add(-7 * 24 * time.Hour).Unix()
	case Range30d:
		return now.Add(-30 * 24 * time.Hour).Unix()
	case RangeAll:
		return 0
	}
	return 0
}

// DisplayNameSQL：COALESCE(NULLIF(display_name,''), username) — 所有榜单的 name 取法
const DisplayNameSQL = "COALESCE(NULLIF(u.display_name,''), u.username)"

// FormatUSD 把 token-quota 转成 "$1,234.56"。Quota / 500000 = USD。
func FormatUSD(quota int64) string {
	dollars := float64(quota) / 500000.0
	return FormatMoney(dollars)
}

// FormatMoney 把美元 float 转成 "$1,234.56"。
func FormatMoney(d float64) string {
	intPart := int64(d)
	cents := int64((d - float64(intPart)) * 100)
	if cents < 0 {
		cents = -cents
	}
	intStr := withThousandSep(intPart)
	if cents < 10 {
		return "$" + intStr + ".0" + strconv.FormatInt(cents, 10)
	}
	return "$" + intStr + "." + strconv.FormatInt(cents, 10)
}

// FormatCount 整数加千分位，例如 12345 → "12,345"。
func FormatCount(n int64) string { return withThousandSep(n) }

func withThousandSep(n int64) string {
	if n < 0 {
		return "-" + withThousandSep(-n)
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b []byte
	first := len(s) % 3
	if first > 0 {
		b = append(b, s[:first]...)
		b = append(b, ',')
	}
	for i := first; i < len(s); i += 3 {
		b = append(b, s[i:i+3]...)
		b = append(b, ',')
	}
	return string(b[:len(b)-1])
}
