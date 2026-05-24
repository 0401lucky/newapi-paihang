package db

import (
	"context"
	"database/sql"
	"time"
)

// Health 在 2 秒超时内对 MySQL 执行一次 Ping，用于 /healthz 路由。
func Health(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}
