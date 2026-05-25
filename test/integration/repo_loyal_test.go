//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
)

func TestLoyal_ClaudeFanReturnsClaude(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.LoyalTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1}, LoyalThreshold: 0.8, LoyalMinCalls: 10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, items)
	var user3Found bool
	for _, it := range items {
		if it.UserID == 3 {
			user3Found = true
			assert.GreaterOrEqual(t, it.Value, 0.8)
			extra, ok := it.Extra.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "claude-sonnet-4", extra["model"])
		}
	}
	assert.True(t, user3Found)
}

func TestLoyal_TwinFanReturnsGemini(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.LoyalTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 10,
		LoyalThreshold: 0.8, LoyalMinCalls: 10,
	})
	require.NoError(t, err)
	var user9Found bool
	for _, it := range items {
		if it.UserID == 9 {
			user9Found = true
			extra := it.Extra.(map[string]any)
			assert.Equal(t, "gemini-2.5-pro", extra["model"])
		}
	}
	assert.True(t, user9Found)
}

func TestLoyal_FiltersSmallSample(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.LoyalTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 100,
		LoyalThreshold: 0.8, LoyalMinCalls: 10,
	})
	require.NoError(t, err)
	for _, it := range items {
		assert.NotEqual(t, int64(8), it.UserID, "user 8 调用 < 10")
	}
}

func TestLoyal_HighThresholdFiltersOut(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.LoyalTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 100,
		LoyalThreshold: 0.99, LoyalMinCalls: 10,
	})
	require.NoError(t, err)
	for _, it := range items {
		assert.NotEqual(t, int64(3), it.UserID)
	}
}
