//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
)

func TestRich_TopOrder(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, total, err := repo.RichTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 5,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 9)
	require.GreaterOrEqual(t, len(items), 5)
	// admin (user_id=1) 余额最高
	assert.Equal(t, int64(1), items[0].UserID)
	assert.Equal(t, "管理员", items[0].Name)
	// 第 2 名 user 2 咸鱼想躺平
	assert.Equal(t, int64(2), items[1].UserID)
}

func TestRich_HiddenUserIDs(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.RichTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 5,
		HiddenUserIDs: []int64{1},
	})
	require.NoError(t, err)
	for _, it := range items {
		assert.NotEqual(t, int64(1), it.UserID, "admin 应被隐藏")
	}
	assert.Equal(t, int64(2), items[0].UserID)
}

func TestRich_ExcludesDisabledAndDeleted(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.RichTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 100,
	})
	require.NoError(t, err)
	for _, it := range items {
		assert.NotEqual(t, int64(10), it.UserID, "禁用用户 10 不应出现")
		assert.NotEqual(t, int64(11), it.UserID, "软删用户 11 不应出现")
	}
}

func TestRich_NameFallbackToUsername(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.RichTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 100,
	})
	require.NoError(t, err)
	var found bool
	for _, it := range items {
		if it.UserID == 8 {
			assert.Equal(t, "plain", it.Name)
			found = true
		}
	}
	assert.True(t, found)
}

func TestRich_RankAssigned(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.RichTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 3,
	})
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.Equal(t, 1, items[0].Rank)
	assert.Equal(t, 2, items[1].Rank)
	assert.Equal(t, 3, items[2].Rank)
}
