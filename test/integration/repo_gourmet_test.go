//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
)

func TestGourmet_30d(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.GourmetTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1},
	})
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.Equal(t, int64(7), items[0].UserID)
	assert.EqualValues(t, 7.0, items[0].Value)
	assert.Equal(t, "7 种", items[0].ValueDisplay)
}
