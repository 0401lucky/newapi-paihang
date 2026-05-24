package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_MissingDSN(t *testing.T) {
	t.Setenv("MYSQL_DSN", "")
	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MYSQL_DSN")
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("MYSQL_DSN", "user:pwd@tcp(host:3306)/db")
	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, 60, cfg.CacheTTLUsers)
	assert.Equal(t, 300, cfg.CacheTTLLogs)
	assert.InDelta(t, 0.8, cfg.LoyalThreshold, 0.001)
	assert.Equal(t, 10, cfg.LoyalMinCalls)
	assert.Equal(t, 200, cfg.RateLimitPerMin)
	assert.Empty(t, cfg.HiddenUserIDs)
	assert.Equal(t, "NewAPI 排行榜", cfg.SiteName)
	assert.Equal(t, []string{"rich", "foodie", "nightowl"}, cfg.EmbedTabsDefault)
}

func TestLoad_HiddenUserIDsParse(t *testing.T) {
	t.Setenv("MYSQL_DSN", "x")
	t.Setenv("HIDDEN_USER_IDS", "1, 42, 99")
	cfg, _ := Load()
	assert.Equal(t, []int64{1, 42, 99}, cfg.HiddenUserIDs)
}

func TestLoad_EmbedTabsParse(t *testing.T) {
	t.Setenv("MYSQL_DSN", "x")
	t.Setenv("EMBED_TABS_DEFAULT", "rich,topup")
	cfg, _ := Load()
	assert.Equal(t, []string{"rich", "topup"}, cfg.EmbedTabsDefault)
}

func TestLoad_AdminTokenOptional(t *testing.T) {
	t.Setenv("MYSQL_DSN", "x")
	cfg, _ := Load()
	assert.False(t, cfg.AdminEnabled())
	t.Setenv("ADMIN_TOKEN", "secret")
	cfg, _ = Load()
	assert.True(t, cfg.AdminEnabled())
	assert.Equal(t, "secret", cfg.AdminToken)
}
