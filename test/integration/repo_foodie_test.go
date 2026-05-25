//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0401lucky/newapi-paihang/internal/repo"
)

func TestFoodie_7d(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, total, err := repo.FoodieTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range7d, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1},
	})
	require.NoError(t, err)
	assert.Greater(t, total, 0)
	require.NotEmpty(t, items)
	// user 3 有 15+1 条 = 16 条 logs（7d 内），应是榜首
	assert.Equal(t, int64(3), items[0].UserID)
	assert.Greater(t, items[0].Value, 10.0)
}

func TestFoodie_Today(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.FoodieTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeToday, Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	if len(items) > 0 {
		assert.GreaterOrEqual(t, items[0].Value, 1.0)
	}
}

func TestFoodie_ValueDisplay(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, _ := repo.FoodieTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range7d, Page: 1, PageSize: 1,
	})
	require.NotEmpty(t, items)
	assert.Contains(t, items[0].ValueDisplay, "次")
}
