package db

import (
	"database/sql"
	"fmt"
	"strings"
)

var requiredColumns = map[string][]string{
	"users":   {"id", "username", "display_name", "status", "quota", "used_quota", "request_count", "created_at", "deleted_at"},
	"logs":    {"id", "user_id", "created_at", "type", "model_name", "quota", "prompt_tokens", "completion_tokens"},
	"top_ups": {"id", "user_id", "money", "status", "create_time"},
}

// CheckSchema 校验上游 NewAPI 库表是否包含本项目依赖的字段。
func CheckSchema(db *sql.DB) error {
	var missing []string
	for table, cols := range requiredColumns {
		rows, err := db.Query("SHOW COLUMNS FROM " + table)
		if err != nil {
			return fmt.Errorf("SHOW COLUMNS FROM %s: %w (该表可能不存在或权限不足)", table, err)
		}
		got := make(map[string]bool)
		for rows.Next() {
			var name, ctype, null, key string
			var def, extra sql.NullString
			if err := rows.Scan(&name, &ctype, &null, &key, &def, &extra); err == nil {
				got[name] = true
			}
		}
		_ = rows.Close()
		for _, c := range cols {
			if !got[c] {
				missing = append(missing, table+"."+c)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("schema 不兼容，缺失字段: %s", strings.Join(missing, ", "))
	}
	return nil
}
