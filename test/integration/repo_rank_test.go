//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/0401lucky/newapi-paihang/internal/repo"
)

func TestSearchUsers_ByDisplayName(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	users, err := repo.SearchUsers(context.Background(), db, "克吹", []int64{1}, 10)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	assert.Equal(t, int64(3), users[0].ID)
	assert.Equal(t, "克吹本吹", users[0].Name)
}

func TestSearchUsers_ByUsername(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	users, err := repo.SearchUsers(context.Background(), db, "plain", nil, 10)
	require.NoError(t, err)
	require.NotEmpty(t, users)
	assert.Equal(t, int64(8), users[0].ID)
	assert.Equal(t, "plain", users[0].Name)
}

func TestSearchUsers_NotFound(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	users, err := repo.SearchUsers(context.Background(), db, "不存在的用户", nil, 10)
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestSearchUsers_ExcludesHidden(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	users, err := repo.SearchUsers(context.Background(), db, "管理员", []int64{1}, 10)
	require.NoError(t, err)
	assert.Empty(t, users)
}

func TestRichRankOf(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	rank, err := repo.RichRankOf(context.Background(), db, 2, []int64{1})
	require.NoError(t, err)
	assert.Equal(t, 1, rank)
}

func TestRichRankOf_NotEligible(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	rank, err := repo.RichRankOf(context.Background(), db, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, rank)
}
