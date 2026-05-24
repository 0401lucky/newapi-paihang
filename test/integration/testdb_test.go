//go:build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedDB_HasSeedData(t *testing.T) {
	db, _ := SharedDB(t)
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE status=1 AND deleted_at IS NULL").Scan(&n)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, 9, "seed 至少 9 个启用且未删除的用户")

	err = db.QueryRow("SELECT COUNT(*) FROM logs WHERE type=2").Scan(&n)
	require.NoError(t, err)
	assert.Greater(t, n, 30, "seed 至少 30 条消费日志")

	err = db.QueryRow("SELECT COUNT(*) FROM top_ups WHERE status='success'").Scan(&n)
	require.NoError(t, err)
	assert.Equal(t, 4, n, "成功充值 4 条")
}
