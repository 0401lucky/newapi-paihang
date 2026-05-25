package persist

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminStore_AddRemoveHidden(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.json")
	store, err := NewAdminStore(path)
	require.NoError(t, err)
	assert.Empty(t, store.HiddenUserIDs())

	require.NoError(t, store.AddHidden(42))
	require.NoError(t, store.AddHidden(99))
	require.NoError(t, store.AddHidden(42)) // 重复
	assert.ElementsMatch(t, []int64{42, 99}, store.HiddenUserIDs())

	require.NoError(t, store.RemoveHidden(42))
	assert.Equal(t, []int64{99}, store.HiddenUserIDs())
}

func TestAdminStore_PersistsAcrossReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.json")
	s1, _ := NewAdminStore(path)
	_ = s1.AddHidden(7)
	s2, err := NewAdminStore(path)
	require.NoError(t, err)
	assert.Equal(t, []int64{7}, s2.HiddenUserIDs())
}
