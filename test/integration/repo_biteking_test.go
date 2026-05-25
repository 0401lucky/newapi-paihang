//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0401lucky/newapi-paihang/internal/repo"
)

func TestBiteking_30d(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.BitekingTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1},
	})
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.Equal(t, int64(6), items[0].UserID)
	assert.EqualValues(t, 5000000.0, items[0].Value)
	extra, ok := items[0].Extra.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "claude-opus-4", extra["model"])
}
