//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0401lucky/newapi-paihang/internal/repo"
)

func TestSpender_All(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.SpenderTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1},
	})
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.Equal(t, int64(2), items[0].UserID) // used_quota 最高
	assert.Greater(t, items[0].Value, 0.0)
}

func TestSpender_7d(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.SpenderTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range7d, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1},
	})
	require.NoError(t, err)
	require.NotEmpty(t, items)
	found := false
	for _, it := range items {
		if it.UserID == 6 {
			found = true
		}
	}
	assert.True(t, found, "user 6 应在 7 天窗口内（单笔王 5M）")
}

func TestSpender_Pagination(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	page1, total, _ := repo.SpenderTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 3,
	})
	page2, _, _ := repo.SpenderTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 2, PageSize: 3,
	})
	assert.Len(t, page1, 3)
	assert.Greater(t, total, 3)
	if len(page2) > 0 {
		assert.NotEqual(t, page1[0].UserID, page2[0].UserID)
		assert.Equal(t, 4, page2[0].Rank)
	}
}
