//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
)

func TestTopup_AllTime(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, total, err := repo.TopupTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total, "只有 user 2/3/4 有成功充值")
	require.GreaterOrEqual(t, len(items), 3)
	// user 2 $4000
	assert.Equal(t, int64(2), items[0].UserID)
	assert.InDelta(t, 4000.0, items[0].Value, 0.01)
	// user 3 共 $3000
	assert.Equal(t, int64(3), items[1].UserID)
	assert.InDelta(t, 3000.0, items[1].Value, 0.01)
}

func TestTopup_FiltersNonSuccess(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.TopupTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 100,
	})
	require.NoError(t, err)
	for _, it := range items {
		assert.NotEqual(t, int64(5), it.UserID, "user 5 充值 pending，不应入榜")
	}
}

func TestTopup_7d(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.TopupTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range7d, Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	for _, it := range items {
		assert.NotEqual(t, int64(3), it.UserID, "user 3 充值都超过 7 天")
	}
}
