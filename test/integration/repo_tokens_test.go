//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
)

func TestTokens_30d(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.TokensTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1},
	})
	require.NoError(t, err)
	require.NotEmpty(t, items)
	ids := map[int64]bool{}
	for i := 0; i < 3 && i < len(items); i++ {
		ids[items[i].UserID] = true
	}
	assert.True(t, ids[6] || ids[3], "user 3 或 6 应在 top3")
}

func TestTokens_ValueDisplay(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, _ := repo.TokensTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 1,
	})
	require.NotEmpty(t, items)
	assert.Contains(t, items[0].ValueDisplay, "tok")
}
