package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// Config 持有所有运行时配置项。
type Config struct {
	MySQLDSN          string
	Port              string
	SiteName          string
	SiteURL           string
	AdminToken        string
	HiddenUserIDs     []int64
	CacheTTLUsers     int // 秒
	CacheTTLLogs      int // 秒
	LoyalThreshold    float64
	LoyalMinCalls     int
	EmbedTabsDefault  []string
	MySQLMaxOpen      int
	MySQLMaxIdle      int
	MySQLConnLifetime int // 秒
	RateLimitPerMin   int
	LogLevel          string
}

// AdminEnabled 当 ADMIN_TOKEN 已配置时返回 true，未配置则禁用 /admin 路由。
func (c *Config) AdminEnabled() bool { return c.AdminToken != "" }

// Load 从环境变量解析配置，并应用默认值。MYSQL_DSN 未设置时返回错误。
func Load() (*Config, error) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		return nil, errors.New("MYSQL_DSN 未设置，参考 .env.example")
	}
	cfg := &Config{
		MySQLDSN:          dsn,
		Port:              getEnv("PORT", "8080"),
		SiteName:          getEnv("SITE_NAME", "NewAPI 排行榜"),
		SiteURL:           os.Getenv("SITE_URL"),
		AdminToken:        os.Getenv("ADMIN_TOKEN"),
		HiddenUserIDs:     parseInt64List(os.Getenv("HIDDEN_USER_IDS")),
		CacheTTLUsers:     getEnvInt("CACHE_TTL_USERS", 60),
		CacheTTLLogs:      getEnvInt("CACHE_TTL_LOGS", 300),
		LoyalThreshold:    getEnvFloat("LOYAL_THRESHOLD", 0.8),
		LoyalMinCalls:     getEnvInt("LOYAL_MIN_CALLS", 10),
		EmbedTabsDefault:  parseStringList(getEnv("EMBED_TABS_DEFAULT", "rich,foodie,nightowl")),
		MySQLMaxOpen:      getEnvInt("MYSQL_MAX_OPEN_CONNS", 10),
		MySQLMaxIdle:      getEnvInt("MYSQL_MAX_IDLE_CONNS", 5),
		MySQLConnLifetime: getEnvInt("MYSQL_CONN_MAX_LIFETIME", 300),
		RateLimitPerMin:   getEnvInt("RATE_LIMIT_PER_MIN", 200),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}
	return cfg, nil
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getEnvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvFloat(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func parseInt64List(s string) []int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.ParseInt(p, 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func parseStringList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
