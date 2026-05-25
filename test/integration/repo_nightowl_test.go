//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0401lucky/newapi-paihang/internal/repo"
)

func TestNightowl_30d(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.NightowlTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1},
	})
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.Equal(t, int64(5), items[0].UserID)
	assert.GreaterOrEqual(t, items[0].Value, 5.0)
	assert.Contains(t, items[0].ValueDisplay, "次")
}
