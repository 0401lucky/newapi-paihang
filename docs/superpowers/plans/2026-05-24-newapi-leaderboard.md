# NewAPI 排行榜 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个独立部署的 NewAPI 排行榜服务，直连 NewAPI MySQL 数据库做只读聚合，提供主站 + 嵌入版 widget + 管理面板三个入口，部署到 Zeabur。

**Architecture:** Go + Gin 后端做 HTTP API 和静态资源分发；Vite 三入口（main/embed/admin）打包 React 前端，通过 `go:embed` 嵌入到 Go 二进制；多阶段 Dockerfile 产出单镜像；内存缓存（sync.Map + singleflight）。

**Tech Stack:** Go 1.22 / Gin / go-sql-driver/mysql / golang.org/x/sync/singleflight / Vite 5 / React 18 / TypeScript / Tailwind CSS 3 / TanStack Query v5 / pnpm / Alpine Docker

**Spec:** `docs/superpowers/specs/2026-05-24-newapi-leaderboard-design.md`

---

## Phase 概览

| Phase | 任务范围 |
|---|---|
| Phase 0 | 项目骨架与基础配置（go mod、目录、.gitignore） |
| Phase 1 | 后端基础设施（config / db / cache / singleflight） |
| Phase 2 | Repo 层 9 个榜单 + 个人查询 SQL（含集成测试） |
| Phase 3 | Service + Middleware + Handler |
| Phase 4 | 静态资源分发 + main.go 装配 + 端到端冒烟 |
| Phase 5 | 前端：脚手架 + 共用组件 + Hooks |
| Phase 6 | 前端：主站 / 嵌入版 / 后台三个入口 |
| Phase 7 | Docker / CI / 部署文档 / 验收 |

每个 Task 走 TDD：写测试 → 跑测试看失败 → 实现 → 跑测试看通过 → 提交。

---

## Phase 0：项目骨架

### Task 0.1：初始化 Go module 与目录结构

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `LICENSE`
- Create: `README.md`（占位，最终在 Phase 7 完善）
- Create: `cmd/leaderboard/.gitkeep`
- Create: `internal/{config,db,cache,repo,service,handler,middleware,persist,embed}/.gitkeep`
- Create: `web/.gitkeep`
- Create: `test/{unit,integration,fixtures}/.gitkeep`
- Create: `data/.gitkeep`
- Create: `docs/.gitkeep`
- Create: `.github/workflows/.gitkeep`

- [ ] **Step 1: 初始化 git 仓库与 go module**

```bash
cd "D:/code/Claude code program/newapi排行榜"
git init -b main
go mod init github.com/yourname/newapi-leaderboard
```

Expected: `go.mod created`

- [ ] **Step 2: 写 .gitignore**

```gitignore
# Go
/leaderboard
/bin/
*.exe
*.test
*.out
coverage.txt

# Node / Vite
node_modules/
web/dist/
.pnpm-store/

# Env / runtime
.env
.env.local
/data/*
!/data/.gitkeep

# IDE
.vscode/
.idea/
*.swp
.DS_Store

# Superpowers 工作目录
.superpowers/
```

- [ ] **Step 3: 创建所有目录与 .gitkeep**

```bash
mkdir -p cmd/leaderboard internal/{config,db,cache,repo,service,handler,middleware,persist,embed} \
         web test/{unit,integration,fixtures} data docs .github/workflows
for d in cmd/leaderboard internal/config internal/db internal/cache internal/repo \
         internal/service internal/handler internal/middleware internal/persist internal/embed \
         web test/unit test/integration test/fixtures data .github/workflows; do
  touch "$d/.gitkeep"
done
```

- [ ] **Step 4: 写最小 README 占位**

`README.md`:
```markdown
# NewAPI 排行榜

NewAPI 用户外部排行榜服务。详见 `docs/superpowers/specs/2026-05-24-newapi-leaderboard-design.md`。

部署文档在 Phase 7 完善。
```

- [ ] **Step 5: 写 MIT LICENSE**

`LICENSE`：标准 MIT 模板，年份 2026，"your name" 占位。

- [ ] **Step 6: 提交**

```bash
git add .
git commit -m "chore: init project skeleton"
```

---

### Task 0.2：安装核心依赖

**Files:**
- Modify: `go.mod`
- Create: `go.sum`

- [ ] **Step 1: 安装 Go 依赖**

```bash
go get github.com/gin-gonic/gin@v1.10.0
go get github.com/go-sql-driver/mysql@v1.8.1
go get golang.org/x/sync@v0.7.0
go get golang.org/x/time@v0.5.0
go get github.com/stretchr/testify@v1.9.0
go get github.com/ory/dockertest/v3@v3.10.0
go mod tidy
```

- [ ] **Step 2: 提交**

```bash
git add go.mod go.sum
git commit -m "chore: add core go dependencies"
```

---

## Phase 1：后端基础设施

### Task 1.1：config 包

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: 写测试**

`internal/config/config_test.go`:
```go
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
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./internal/config/...
```
Expected: 编译失败 "undefined: Load"

- [ ] **Step 3: 实现 config**

`internal/config/config.go`:
```go
package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	MySQLDSN          string
	Port              string
	SiteName          string
	SiteURL           string
	AdminToken        string
	HiddenUserIDs     []int64
	CacheTTLUsers     int
	CacheTTLLogs      int
	LoyalThreshold    float64
	LoyalMinCalls     int
	EmbedTabsDefault  []string
	MySQLMaxOpen      int
	MySQLMaxIdle      int
	MySQLConnLifetime int
	RateLimitPerMin   int
	LogLevel          string
}

func (c *Config) AdminEnabled() bool { return c.AdminToken != "" }

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
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./internal/config/... -v
```
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/config/
git commit -m "feat(config): env-driven config with sensible defaults"
```

---

### Task 1.2：db 包（连接池 + 健康检查 + Schema 校验）

**Files:**
- Create: `internal/db/mysql.go`
- Create: `internal/db/schema_check.go`
- Create: `internal/db/health.go`
- Test: `internal/db/schema_check_test.go`（unit，无需真实 DB）
- Test: `test/integration/db_test.go`（集成，需 dockertest）

- [ ] **Step 1: 写 schema_check 的 unit 测试（用 sqlmock 风格的 fake driver）**

为简化，schema_check 单测放到集成测试 Phase 一起，这里只先写 mysql.go 的 OpenPool 函数。

`internal/db/mysql.go`:
```go
package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/yourname/newapi-leaderboard/internal/config"
)

func OpenPool(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetMaxOpenConns(cfg.MySQLMaxOpen)
	db.SetMaxIdleConns(cfg.MySQLMaxIdle)
	db.SetConnMaxLifetime(time.Duration(cfg.MySQLConnLifetime) * time.Second)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db.Ping: %w", err)
	}
	return db, nil
}
```

- [ ] **Step 2: 实现 schema_check.go**

`internal/db/schema_check.go`:
```go
package db

import (
	"database/sql"
	"fmt"
	"strings"
)

var requiredColumns = map[string][]string{
	"users":   {"id", "username", "display_name", "status", "quota", "used_quota", "request_count", "created_at", "deleted_at"},
	"logs":    {"id", "user_id", "created_at", "type", "model_name", "quota", "prompt_tokens", "completion_tokens"},
	"top_ups": {"id", "user_id", "money", "status", "create_time"},
}

func CheckSchema(db *sql.DB) error {
	var missing []string
	for table, cols := range requiredColumns {
		rows, err := db.Query("SHOW COLUMNS FROM " + table)
		if err != nil {
			return fmt.Errorf("SHOW COLUMNS FROM %s: %w (该表可能不存在或权限不足)", table, err)
		}
		got := make(map[string]bool)
		for rows.Next() {
			var name, ctype, null, key string
			var def, extra sql.NullString
			if err := rows.Scan(&name, &ctype, &null, &key, &def, &extra); err == nil {
				got[name] = true
			}
		}
		_ = rows.Close()
		for _, c := range cols {
			if !got[c] {
				missing = append(missing, table+"."+c)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("schema 不兼容，缺失字段: %s", strings.Join(missing, ", "))
	}
	return nil
}
```

- [ ] **Step 3: 实现 health.go**

`internal/db/health.go`:
```go
package db

import (
	"context"
	"database/sql"
	"time"
)

func Health(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}
```

- [ ] **Step 4: 跑 build 看通过**

```bash
go build ./...
```
Expected: 编译通过

- [ ] **Step 5: 提交**

```bash
git add internal/db/
git commit -m "feat(db): mysql pool with schema validation and health check"
```

> 集成测试在 Task 2.0 一起做（dockertest 启 MySQL + seed.sql）。

---

### Task 1.3：cache 包（sync.Map + TTL）

**Files:**
- Create: `internal/cache/memory.go`
- Test: `internal/cache/memory_test.go`

- [ ] **Step 1: 写测试**

`internal/cache/memory_test.go`:
```go
package cache

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetGet_Basic(t *testing.T) {
	c := New()
	c.Set("a", "hello", 1*time.Second)
	v, ok, stale := c.Get("a")
	require.True(t, ok)
	assert.Equal(t, "hello", v)
	assert.False(t, stale)
}

func TestGet_Miss(t *testing.T) {
	c := New()
	_, ok, _ := c.Get("missing")
	assert.False(t, ok)
}

func TestExpire_BecomesStale(t *testing.T) {
	c := New()
	c.Set("a", 123, 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	v, ok, stale := c.Get("a")
	require.True(t, ok, "过期数据仍可作为 stale 返回")
	assert.Equal(t, 123, v)
	assert.True(t, stale)
}

func TestDelete(t *testing.T) {
	c := New()
	c.Set("a", "x", time.Second)
	c.Delete("a")
	_, ok, _ := c.Get("a")
	assert.False(t, ok)
}

func TestDeletePrefix(t *testing.T) {
	c := New()
	c.Set("lb:rich:p1", 1, time.Second)
	c.Set("lb:rich:p2", 2, time.Second)
	c.Set("lb:foodie:p1", 3, time.Second)
	n := c.DeletePrefix("lb:rich:")
	assert.Equal(t, 2, n)
	_, ok, _ := c.Get("lb:rich:p1")
	assert.False(t, ok)
	_, ok, _ = c.Get("lb:foodie:p1")
	assert.True(t, ok)
}

func TestClear(t *testing.T) {
	c := New()
	c.Set("a", 1, time.Second)
	c.Set("b", 2, time.Second)
	c.Clear()
	_, ok, _ := c.Get("a")
	assert.False(t, ok)
}

func TestStats(t *testing.T) {
	c := New()
	c.Set("a", 1, time.Second)
	_, _, _ = c.Get("a")     // hit
	_, _, _ = c.Get("miss")  // miss
	s := c.Stats()
	assert.EqualValues(t, 1, s.Hits)
	assert.EqualValues(t, 1, s.Misses)
}

func TestConcurrentSafety(t *testing.T) {
	c := New()
	var wg sync.WaitGroup
	var done int32
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				c.Set("k", j, time.Second)
				_, _, _ = c.Get("k")
			}
			atomic.AddInt32(&done, 1)
		}(i)
	}
	wg.Wait()
	assert.EqualValues(t, 50, done)
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./internal/cache/...
```
Expected: 编译失败 "undefined: New"

- [ ] **Step 3: 实现 memory.go**

`internal/cache/memory.go`:
```go
package cache

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type entry struct {
	val       any
	expiresAt time.Time
}

type Cache struct {
	mu     sync.RWMutex
	data   map[string]entry
	hits   atomic.Int64
	misses atomic.Int64
}

type Stats struct {
	Hits, Misses int64
	Size         int
}

func New() *Cache {
	return &Cache{data: make(map[string]entry)}
}

func (c *Cache) Set(key string, val any, ttl time.Duration) {
	c.mu.Lock()
	c.data[key] = entry{val: val, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

// Get 返回 (val, ok, stale)。ok=true 时仍可能 stale=true（过期但保留作 fallback）。
func (c *Cache) Get(key string) (any, bool, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		c.misses.Add(1)
		return nil, false, false
	}
	c.hits.Add(1)
	stale := time.Now().After(e.expiresAt)
	return e.val, true, stale
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
}

func (c *Cache) DeletePrefix(prefix string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for k := range c.data {
		if strings.HasPrefix(k, prefix) {
			delete(c.data, k)
			n++
		}
	}
	return n
}

func (c *Cache) Clear() {
	c.mu.Lock()
	c.data = make(map[string]entry)
	c.mu.Unlock()
}

func (c *Cache) Stats() Stats {
	c.mu.RLock()
	size := len(c.data)
	c.mu.RUnlock()
	return Stats{Hits: c.hits.Load(), Misses: c.misses.Load(), Size: size}
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./internal/cache/... -v -race
```
Expected: 全部 PASS，无 race condition

- [ ] **Step 5: 提交**

```bash
git add internal/cache/
git commit -m "feat(cache): in-memory cache with TTL + stale fallback + stats"
```

---

### Task 1.4：cache 包增加 singleflight 包装

**Files:**
- Create: `internal/cache/singleflight.go`
- Test: `internal/cache/singleflight_test.go`

- [ ] **Step 1: 写测试**

`internal/cache/singleflight_test.go`:
```go
package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrLoad_FirstCallFetches(t *testing.T) {
	c := New()
	calls := atomic.Int32{}
	v, stale, err := c.GetOrLoad("k", time.Second, func() (any, error) {
		calls.Add(1)
		return 42, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 42, v)
	assert.False(t, stale)
	assert.EqualValues(t, 1, calls.Load())
}

func TestGetOrLoad_CacheHitSkipsLoader(t *testing.T) {
	c := New()
	c.Set("k", "cached", time.Second)
	calls := atomic.Int32{}
	v, _, err := c.GetOrLoad("k", time.Second, func() (any, error) {
		calls.Add(1)
		return "loaded", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "cached", v)
	assert.EqualValues(t, 0, calls.Load())
}

func TestGetOrLoad_Singleflight(t *testing.T) {
	c := New()
	calls := atomic.Int32{}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = c.GetOrLoad("k", time.Second, func() (any, error) {
				calls.Add(1)
				time.Sleep(50 * time.Millisecond)
				return "ok", nil
			})
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, calls.Load(), "20 个并发只触发 1 次 loader")
}

func TestGetOrLoad_StaleFallback(t *testing.T) {
	c := New()
	c.Set("k", "old", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	v, stale, err := c.GetOrLoad("k", time.Second, func() (any, error) {
		return nil, errors.New("db down")
	})
	require.NoError(t, err, "loader 失败但有 stale 数据时不报错")
	assert.Equal(t, "old", v)
	assert.True(t, stale)
}

func TestGetOrLoad_LoaderErrorNoStale(t *testing.T) {
	c := New()
	_, _, err := c.GetOrLoad("k", time.Second, func() (any, error) {
		return nil, errors.New("boom")
	})
	require.Error(t, err)
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./internal/cache/...
```
Expected: 编译失败 "undefined: (*Cache).GetOrLoad"

- [ ] **Step 3: 实现 singleflight.go**

`internal/cache/singleflight.go`:
```go
package cache

import (
	"time"

	"golang.org/x/sync/singleflight"
)

var sfGroup singleflight.Group

// GetOrLoad 是缓存的核心入口：
// 1. 缓存命中且未过期 → 直接返回
// 2. 缓存命中但过期 → 调用 loader 刷新；loader 失败时返回旧数据（stale=true）
// 3. 缓存未命中 → singleflight 合并并发请求，调用 loader 一次
func (c *Cache) GetOrLoad(key string, ttl time.Duration, loader func() (any, error)) (any, bool, error) {
	if v, ok, stale := c.Get(key); ok && !stale {
		return v, false, nil
	}
	result, err, _ := sfGroup.Do(key, func() (any, error) {
		val, err := loader()
		if err != nil {
			// loader 失败，但如果有旧值就返回旧值
			if old, ok, _ := c.Get(key); ok {
				return staleResult{val: old}, nil
			}
			return nil, err
		}
		c.Set(key, val, ttl)
		return val, nil
	})
	if err != nil {
		return nil, false, err
	}
	if s, ok := result.(staleResult); ok {
		return s.val, true, nil
	}
	return result, false, nil
}

type staleResult struct{ val any }
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./internal/cache/... -v -race
```
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/cache/
git commit -m "feat(cache): GetOrLoad with singleflight + stale-while-error"
```

---

## Phase 2：Repo 层 9 个榜单 + 个人查询

### Task 2.0：集成测试基础（dockertest + 共享 fixtures）

> Phase 2 所有 repo 测试都基于 MySQL 真实查询，用 dockertest 启动临时 MySQL，灌入 `test/fixtures/seed.sql` 后跑断言。这个 task 一次性把脚手架搭好。

**Files:**
- Create: `test/fixtures/seed.sql`
- Create: `test/integration/testdb.go`
- Create: `test/integration/testdb_test.go`

- [ ] **Step 1: 写种子数据 seed.sql（10 用户 + 200 logs + 5 充值，覆盖各种边界）**

`test/fixtures/seed.sql`:
```sql
-- 模拟 NewAPI 表结构（只含本项目用到的字段）
CREATE TABLE IF NOT EXISTS users (
  id            INT PRIMARY KEY,
  username      VARCHAR(64) NOT NULL,
  display_name  VARCHAR(64) DEFAULT '',
  role          INT DEFAULT 1,
  status        INT DEFAULT 1,
  email         VARCHAR(128) DEFAULT '',
  quota         BIGINT DEFAULT 0,
  used_quota    BIGINT DEFAULT 0,
  request_count INT DEFAULT 0,
  `group`       VARCHAR(64) DEFAULT 'default',
  aff_code      VARCHAR(32) DEFAULT '',
  aff_count     INT DEFAULT 0,
  aff_quota     BIGINT DEFAULT 0,
  aff_history   BIGINT DEFAULT 0,
  inviter_id    INT DEFAULT 0,
  created_at    BIGINT NOT NULL,
  last_login_at BIGINT DEFAULT 0,
  deleted_at    DATETIME NULL,
  INDEX idx_status_deleted (status, deleted_at)
);

CREATE TABLE IF NOT EXISTS logs (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id             INT NOT NULL,
  created_at          BIGINT NOT NULL,
  type                INT NOT NULL,
  content             TEXT,
  username            VARCHAR(64),
  token_name          VARCHAR(64),
  model_name          VARCHAR(128),
  quota               BIGINT DEFAULT 0,
  prompt_tokens       INT DEFAULT 0,
  completion_tokens   INT DEFAULT 0,
  use_time            INT DEFAULT 0,
  is_stream           BOOLEAN DEFAULT FALSE,
  channel             INT,
  token_id            INT,
  `group`             VARCHAR(64),
  ip                  VARCHAR(64),
  request_id          VARCHAR(64),
  upstream_request_id VARCHAR(64),
  other               TEXT,
  INDEX idx_user (user_id),
  INDEX idx_created (created_at),
  INDEX idx_model (model_name),
  INDEX idx_created_type (created_at, type)
);

CREATE TABLE IF NOT EXISTS top_ups (
  id              INT PRIMARY KEY AUTO_INCREMENT,
  user_id         INT NOT NULL,
  amount          BIGINT,
  money           DOUBLE NOT NULL,
  trade_no        VARCHAR(64),
  payment_method  VARCHAR(32),
  payment_provider VARCHAR(32),
  create_time     BIGINT NOT NULL,
  complete_time   BIGINT,
  status          VARCHAR(32) NOT NULL
);

-- 10 个用户：1=admin（隐藏候选）、2-9=正常、10=被禁用、11=软删除
INSERT INTO users (id, username, display_name, role, status, quota, used_quota, request_count, created_at, deleted_at) VALUES
  (1,  'admin',      '管理员',         100, 1, 999999999999, 0,           0,     1700000000, NULL),
  (2,  'tycoon',     '咸鱼想躺平',     1,   1, 4216060000,   2500000000,  1200,  1700100000, NULL),
  (3,  'clude_fan',  '克吹本吹',       1,   1, 2600940000,   1800000000,  3500,  1700200000, NULL),
  (4,  'sub_clude',  '小克的奴',       1,   1, 1833700000,   1200000000,  800,   1700300000, NULL),
  (5,  'nightowl',   '夜半敲键人',     1,   1, 1054250000,   900000000,   5500,  1700400000, NULL),
  (6,  'coffee',     '咖啡续命',       1,   1, 949500000,    700000000,   2200,  1700500000, NULL),
  (7,  'gourmet',    '万物皆可尝',     1,   1, 500000000,    400000000,   1800,  1700600000, NULL),
  (8,  'plain',      '',               1,   1, 100000000,    50000000,    300,   1700700000, NULL),  -- 无 display_name
  (9,  'twin_fan',   '双子奶妈',       1,   1, 200000000,    600000000,   2400,  1700800000, NULL),
  (10, 'banned',     '已封禁',         1,   2, 99999,        0,           0,     1700900000, NULL),  -- status=2
  (11, 'deleted',    '已删除',         1,   1, 99999,        0,           0,     1701000000, '2026-01-01 00:00:00'); -- 软删

-- logs：覆盖各种 type、时间、模型；时间戳用 2026-05 月内的（约 1779600000+）
-- 假定测试运行时取 NOW() 后回溯，所以这些 created_at 要离 NOW() 近
-- 实际写入时由 testdb helper 用 UNIX_TIMESTAMP() 动态注入
-- 这里用变量占位，testdb 在 ApplySeed 时做替换
SET @NOW := UNIX_TIMESTAMP();

-- user 2: 大消费、claude 重度（死忠粉候选）
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (2, @NOW-3600,     2, 'claude-sonnet-4',  500000, 1000, 2000),
  (2, @NOW-7200,     2, 'claude-sonnet-4',  600000, 1200, 2400),
  (2, @NOW-86400*2,  2, 'claude-sonnet-4',  700000, 1500, 3000),
  (2, @NOW-86400*5,  2, 'claude-opus-4',    900000, 800,  4000);

-- user 3: 大调用次数、纯 claude（死忠粉满足）
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens)
SELECT 3, @NOW - 3600*seq, 2, 'claude-sonnet-4', 100000, 500, 1500
FROM (SELECT 1 AS seq UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9 UNION SELECT 10
      UNION SELECT 11 UNION SELECT 12 UNION SELECT 13 UNION SELECT 14 UNION SELECT 15) t;
-- 加一条非 claude，验证占比仍 >= 0.8
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (3, @NOW-86400, 2, 'gemini-2.5-pro', 50000, 200, 800);

-- user 4: 中等
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (4, @NOW-3600,    2, 'claude-sonnet-4', 200000, 600, 1800),
  (4, @NOW-86400,   2, 'gpt-4o',          150000, 400, 1200),
  (4, @NOW-86400*3, 2, 'gpt-4o',          180000, 500, 1500);

-- user 5: 熬夜冠军（凌晨 UTC+8 调用）
-- UTC+8 凌晨 N 点 = UTC (N+16) mod 24 点。这里 MySQL 默认 timezone=UTC，
-- FROM_UNIXTIME(NOW()) 返回 UTC 时间。所以要插入 UTC 18:00-21:00 对应 UTC+8 02:00-05:00。
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (5, UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(@NOW), '%Y-%m-%d 19:00:00')) - 86400*1, 2, 'claude-sonnet-4', 100000, 300, 800),  -- UTC+8 03:00
  (5, UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(@NOW), '%Y-%m-%d 18:00:00')) - 86400*1, 2, 'gpt-4o',          120000, 400, 900),  -- UTC+8 02:00
  (5, UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(@NOW), '%Y-%m-%d 20:00:00')) - 86400*2, 2, 'claude-opus-4',   200000, 500, 1500), -- UTC+8 04:00
  (5, UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(@NOW), '%Y-%m-%d 17:30:00')) - 86400*3, 2, 'gemini-2.5-pro',  150000, 400, 1200), -- UTC+8 01:30
  (5, UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(@NOW), '%Y-%m-%d 21:00:00')) - 86400*4, 2, 'claude-sonnet-4', 100000, 300, 800);  -- UTC+8 05:00

-- user 6: 单笔王候选
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (6, @NOW-3600,  2, 'claude-opus-4', 5000000, 10000, 20000),  -- 极大单笔
  (6, @NOW-86400, 2, 'claude-sonnet-4', 100000, 300, 800);

-- user 7: 美食家（多模型）
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (7, @NOW-3600,    2, 'claude-sonnet-4', 100000, 200, 600),
  (7, @NOW-7200,    2, 'claude-opus-4',   200000, 400, 1200),
  (7, @NOW-10800,   2, 'gpt-4o',          150000, 300, 900),
  (7, @NOW-14400,   2, 'gpt-4o-mini',     50000,  200, 500),
  (7, @NOW-18000,   2, 'gemini-2.5-pro',  120000, 250, 750),
  (7, @NOW-21600,   2, 'gemini-2.5-flash', 30000, 150, 400),
  (7, @NOW-25200,   2, 'deepseek-chat',    40000, 200, 600);

-- user 8: 无 display_name（验证回退）
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (8, @NOW-3600, 2, 'gpt-4o', 50000, 100, 300);

-- user 9: 双子粉
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens)
SELECT 9, @NOW - 3600*seq, 2, 'gemini-2.5-pro', 80000, 300, 900
FROM (SELECT 1 AS seq UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6
      UNION SELECT 7 UNION SELECT 8 UNION SELECT 9 UNION SELECT 10
      UNION SELECT 11 UNION SELECT 12) t;

-- user 1 (admin): 也有一些消费，验证隐藏功能
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (1, @NOW-3600, 2, 'claude-sonnet-4', 999999, 5000, 10000);

-- top_ups 充值记录
INSERT INTO top_ups (user_id, amount, money, status, create_time) VALUES
  (2, 2000000000, 4000.00, 'success', UNIX_TIMESTAMP() - 86400*3),
  (3, 1000000000, 2000.00, 'success', UNIX_TIMESTAMP() - 86400*10),
  (3, 500000000,  1000.00, 'success', UNIX_TIMESTAMP() - 86400*20),
  (4, 300000000,  600.00,  'success', UNIX_TIMESTAMP() - 86400*5),
  (5, 200000000,  400.00,  'pending', UNIX_TIMESTAMP() - 86400*1);  -- 非 success，应被过滤
```

- [ ] **Step 2: 写 testdb helper**

`test/integration/testdb.go`:
```go
package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	sharedDB     *sql.DB
	sharedDSN    string
	sharedOnce   sync.Once
	sharedPool   *dockertest.Pool
	sharedRes    *dockertest.Resource
	sharedErr    error
)

// SharedDB 启动一次 MySQL 容器，灌入 seed.sql；多个 test 复用。
// 测试退出时由 dockertest 自动清理（设置 ExpireSeconds）。
func SharedDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	sharedOnce.Do(func() {
		sharedDB, sharedDSN, sharedErr = startMySQL()
	})
	if sharedErr != nil {
		t.Fatalf("启动测试 MySQL 失败: %v", sharedErr)
	}
	return sharedDB, sharedDSN
}

func startMySQL() (*sql.DB, string, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, "", fmt.Errorf("dockertest pool: %w", err)
	}
	pool.MaxWait = 60 * time.Second
	res, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mysql",
		Tag:        "8.0",
		Env: []string{
			"MYSQL_ROOT_PASSWORD=root",
			"MYSQL_DATABASE=newapi_test",
		},
		Cmd: []string{"--default-authentication-plugin=mysql_native_password"},
	}, func(hc *docker.HostConfig) {
		hc.AutoRemove = true
		hc.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, "", fmt.Errorf("docker run mysql: %w", err)
	}
	_ = res.Expire(600) // 10 分钟兜底清理

	dsn := fmt.Sprintf("root:root@tcp(localhost:%s)/newapi_test?parseTime=true&multiStatements=true", res.GetPort("3306/tcp"))
	sharedPool, sharedRes = pool, res

	var db *sql.DB
	if err := pool.Retry(func() error {
		var e error
		db, e = sql.Open("mysql", dsn)
		if e != nil {
			return e
		}
		return db.Ping()
	}); err != nil {
		return nil, "", fmt.Errorf("等待 MySQL 就绪: %w", err)
	}

	if err := applySeed(db); err != nil {
		return nil, "", fmt.Errorf("apply seed: %w", err)
	}
	return db, dsn, nil
}

func applySeed(db *sql.DB) error {
	// 找到项目根（含 go.mod）
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(root, "test", "fixtures", "seed.sql"))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = db.ExecContext(ctx, string(data))
	return err
}

func findProjectRoot() (string, error) {
	wd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", fmt.Errorf("找不到 go.mod")
		}
		wd = parent
	}
}

// ResetSeed 在每个测试开始时调用：清表 + 重新灌种子，保证测试互不污染
func ResetSeed(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"logs", "top_ups", "users"} {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			t.Fatalf("清表 %s 失败: %v", table, err)
		}
	}
	if err := applySeed(db); err != nil {
		t.Fatalf("重新灌种子失败: %v", err)
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	_ = strings.HasPrefix // 防 go vet 移除 strings import
}
```

- [ ] **Step 3: 写 testdb 自身的健全性测试**

`test/integration/testdb_test.go`:
```go
//go:build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedDB_HasSeedData(t *testing.T) {
	db, _ := SharedDB(t)
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE status=1 AND deleted_at IS NULL").Scan(&n)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, 9, "seed 至少 9 个启用且未删除的用户")

	err = db.QueryRow("SELECT COUNT(*) FROM logs WHERE type=2").Scan(&n)
	require.NoError(t, err)
	assert.Greater(t, n, 30, "seed 至少 30 条消费日志")

	err = db.QueryRow("SELECT COUNT(*) FROM top_ups WHERE status='success'").Scan(&n)
	require.NoError(t, err)
	assert.Equal(t, 4, n, "成功充值 4 条")
}
```

- [ ] **Step 4: 跑集成测试**

```bash
go test -tags=integration ./test/integration/...
```
Expected: PASS（首次运行约 30-60s 拉镜像 + 启容器）

> 如果机器没装 Docker，跳过这个 task 直到部署前再跑。CI 上 Docker 必备。

- [ ] **Step 5: 提交**

```bash
git add test/fixtures/seed.sql test/integration/
git commit -m "test: shared MySQL fixture with dockertest + seed data"
```

---

### Task 2.1：repo 公共类型与辅助函数

**Files:**
- Create: `internal/repo/repo.go`
- Test: `internal/repo/repo_test.go`

- [ ] **Step 1: 写测试**

`internal/repo/repo_test.go`:
```go
package repo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildHiddenClause(t *testing.T) {
	clause, args := BuildHiddenClause(nil)
	assert.Empty(t, clause)
	assert.Empty(t, args)

	clause, args = BuildHiddenClause([]int64{1, 2, 3})
	assert.Equal(t, " AND u.id NOT IN (?,?,?)", clause)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, args)
}

func TestRangeStart_Today(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	now := time.Date(2026, 5, 24, 15, 30, 0, 0, loc)
	ts := rangeStartAt(RangeToday, now)
	expected := time.Date(2026, 5, 24, 0, 0, 0, 0, loc).Unix()
	assert.Equal(t, expected, ts)
}

func TestRangeStart_7d(t *testing.T) {
	now := time.Date(2026, 5, 24, 15, 30, 0, 0, time.UTC)
	ts := rangeStartAt(Range7d, now)
	assert.Equal(t, now.Add(-7*24*time.Hour).Unix(), ts)
}

func TestRangeStart_All(t *testing.T) {
	now := time.Now()
	ts := rangeStartAt(RangeAll, now)
	assert.EqualValues(t, 0, ts)
}

func TestValidRange(t *testing.T) {
	assert.True(t, ValidRange("today"))
	assert.True(t, ValidRange("7d"))
	assert.True(t, ValidRange("30d"))
	assert.True(t, ValidRange("all"))
	assert.False(t, ValidRange("yesterday"))
	assert.False(t, ValidRange(""))
}

func TestFormatUSD(t *testing.T) {
	assert.Equal(t, "$0.00", FormatUSD(0))
	assert.Equal(t, "$2.00", FormatUSD(1000000))
	assert.Equal(t, "$8,432.12", FormatUSD(4216060000))
}

func TestFormatCount(t *testing.T) {
	assert.Equal(t, "0", FormatCount(0))
	assert.Equal(t, "999", FormatCount(999))
	assert.Equal(t, "1,000", FormatCount(1000))
	assert.Equal(t, "12,345,678", FormatCount(12345678))
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./internal/repo/...
```
Expected: 编译失败

- [ ] **Step 3: 实现 repo.go**

`internal/repo/repo.go`:
```go
package repo

import (
	"strconv"
	"strings"
	"time"
)

// CSTZone 熬夜冠军和 today 计算硬编码 +08:00
var CSTZone = time.FixedZone("CST", 8*3600)

// Range 时间窗类型
type Range string

const (
	RangeToday Range = "today"
	Range7d    Range = "7d"
	Range30d   Range = "30d"
	RangeAll   Range = "all"
)

func ValidRange(r string) bool {
	switch Range(r) {
	case RangeToday, Range7d, Range30d, RangeAll:
		return true
	}
	return false
}

// LeaderboardItem 是 9 个榜单的统一行结构
type LeaderboardItem struct {
	Rank         int     `json:"rank"`
	UserID       int64   `json:"user_id"`
	Name         string  `json:"name"`
	Value        float64 `json:"value"`
	ValueDisplay string  `json:"value_display"`
	Extra        any     `json:"extra,omitempty"`
}

// QueryParams 所有榜单共用的查询参数
type QueryParams struct {
	Range          Range
	Page           int
	PageSize       int
	HiddenUserIDs  []int64
	LoyalThreshold float64 // 仅死忠粉用
	LoyalMinCalls  int     // 仅死忠粉用
}

// Offset 分页偏移
func (q QueryParams) Offset() int { return (q.Page - 1) * q.PageSize }

// BuildHiddenClause 返回 (clause, args)，hidden 为空时返回空串
func BuildHiddenClause(hidden []int64) (string, []any) {
	if len(hidden) == 0 {
		return "", nil
	}
	placeholders := strings.Repeat("?,", len(hidden))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(hidden))
	for i, id := range hidden {
		args[i] = id
	}
	return " AND u.id NOT IN (" + placeholders + ")", args
}

// RangeStart 返回时间窗起始 Unix 秒；RangeAll 返回 0
func RangeStart(r Range) int64 {
	return rangeStartAt(r, time.Now())
}

func rangeStartAt(r Range, now time.Time) int64 {
	switch r {
	case RangeToday:
		cst := now.In(CSTZone)
		t0 := time.Date(cst.Year(), cst.Month(), cst.Day(), 0, 0, 0, 0, CSTZone)
		return t0.Unix()
	case Range7d:
		return now.Add(-7 * 24 * time.Hour).Unix()
	case Range30d:
		return now.Add(-30 * 24 * time.Hour).Unix()
	case RangeAll:
		return 0
	}
	return 0
}

// DisplayNameSQL：COALESCE(NULLIF(display_name,''), username) — 所有榜单的 name 取法
const DisplayNameSQL = "COALESCE(NULLIF(u.display_name,''), u.username)"

// FormatUSD 把 token-quota 转成 "$1,234.56"
func FormatUSD(quota int64) string {
	dollars := float64(quota) / 500000.0
	return FormatMoney(dollars)
}

// FormatMoney 直接传美元 float
func FormatMoney(d float64) string {
	intPart := int64(d)
	cents := int64((d - float64(intPart)) * 100)
	if cents < 0 {
		cents = -cents
	}
	intStr := withThousandSep(intPart)
	if cents < 10 {
		return "$" + intStr + ".0" + strconv.FormatInt(cents, 10)
	}
	return "$" + intStr + "." + strconv.FormatInt(cents, 10)
}

// FormatCount 整数加千分位
func FormatCount(n int64) string { return withThousandSep(n) }

func withThousandSep(n int64) string {
	if n < 0 {
		return "-" + withThousandSep(-n)
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b []byte
	first := len(s) % 3
	if first > 0 {
		b = append(b, s[:first]...)
		b = append(b, ',')
	}
	for i := first; i < len(s); i += 3 {
		b = append(b, s[i:i+3]...)
		b = append(b, ',')
	}
	return string(b[:len(b)-1])
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./internal/repo/... -v
```
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/repo/
git commit -m "feat(repo): shared types, range helpers, format utilities"
```

---

### Task 2.2：土豪榜 repo (rich)

**Files:**
- Create: `internal/repo/rich.go`
- Test: `test/integration/repo_rich_test.go`

- [ ] **Step 1: 写集成测试**

`test/integration/repo_rich_test.go`:
```go
//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
)

func TestRich_TopOrder(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, total, err := repo.RichTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 5,
	})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 9)
	require.GreaterOrEqual(t, len(items), 5)
	// admin (user_id=1) 余额最高但应在结果里（未隐藏的情况）
	assert.Equal(t, int64(1), items[0].UserID)
	assert.Equal(t, "管理员", items[0].Name)
	// 第 2 名应该是 user 2 咸鱼想躺平
	assert.Equal(t, int64(2), items[1].UserID)
}

func TestRich_HiddenUserIDs(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.RichTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 5,
		HiddenUserIDs: []int64{1}, // 隐藏 admin
	})
	require.NoError(t, err)
	for _, it := range items {
		assert.NotEqual(t, int64(1), it.UserID, "admin 应被隐藏")
	}
	assert.Equal(t, int64(2), items[0].UserID, "隐藏 admin 后第一名是咸鱼想躺平")
}

func TestRich_ExcludesDisabledAndDeleted(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.RichTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 100,
	})
	require.NoError(t, err)
	for _, it := range items {
		assert.NotEqual(t, int64(10), it.UserID, "禁用用户 10 不应出现")
		assert.NotEqual(t, int64(11), it.UserID, "软删用户 11 不应出现")
	}
}

func TestRich_NameFallbackToUsername(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.RichTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 100,
	})
	require.NoError(t, err)
	var found bool
	for _, it := range items {
		if it.UserID == 8 {
			assert.Equal(t, "plain", it.Name, "display_name 空时回退 username")
			found = true
		}
	}
	assert.True(t, found, "user 8 应在结果里")
}

func TestRich_RankAssigned(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.RichTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 3,
	})
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.Equal(t, 1, items[0].Rank)
	assert.Equal(t, 2, items[1].Rank)
	assert.Equal(t, 3, items[2].Rank)
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test -tags=integration ./test/integration/... -run TestRich
```
Expected: 编译失败 "undefined: repo.RichTop"

- [ ] **Step 3: 实现 rich.go**

`internal/repo/rich.go`:
```go
package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// RichTop 返回钱包余额最多的用户。无时间维度。
func RichTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)

	countSQL := "SELECT COUNT(*) FROM users u WHERE u.status=1 AND u.deleted_at IS NULL" + hiddenClause
	var total int
	if err := db.QueryRowContext(ctx, countSQL, hiddenArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("rich count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT u.id, %s AS name, u.quota
FROM users u
WHERE u.status=1 AND u.deleted_at IS NULL%s
ORDER BY u.quota DESC, u.id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, hiddenClause)

	args := append(hiddenArgs, q.PageSize, q.Offset())
	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("rich query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var quota int64
		if err := rows.Scan(&it.UserID, &it.Name, &quota); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(quota)
		it.ValueDisplay = FormatUSD(quota)
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./internal/repo/...
go test -tags=integration ./test/integration/... -run TestRich -v
```
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/repo/rich.go test/integration/repo_rich_test.go
git commit -m "feat(repo): rich leaderboard by users.quota"
```

---

### Task 2.3：散财榜 repo (spender)

**Files:**
- Create: `internal/repo/spender.go`
- Test: `test/integration/repo_spender_test.go`

> 散财榜分两种实现：
> - `RangeAll` 用 `users.used_quota`（累计字段，准确）
> - `Range7d/30d/today` 用 `logs SUM(quota) WHERE type=2`（窗口聚合）

- [ ] **Step 1: 写集成测试**

`test/integration/repo_spender_test.go`:
```go
//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
)

func TestSpender_All(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.SpenderTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1}, // 隐藏 admin
	})
	require.NoError(t, err)
	require.NotEmpty(t, items)
	// user 2 used_quota=2500000000，应是最高
	assert.Equal(t, int64(2), items[0].UserID)
	assert.Greater(t, items[0].Value, 0.0)
}

func TestSpender_7d(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.SpenderTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range7d, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1},
	})
	require.NoError(t, err)
	// 7 天窗口下，user 6 因为单笔王巨额消耗应该排很高
	require.NotEmpty(t, items)
	found := false
	for _, it := range items {
		if it.UserID == 6 {
			found = true
		}
	}
	assert.True(t, found, "user 6 应在 7 天窗口内")
}

func TestSpender_Pagination(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	page1, total, _ := repo.SpenderTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 3,
	})
	page2, _, _ := repo.SpenderTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 2, PageSize: 3,
	})
	assert.Len(t, page1, 3)
	assert.Greater(t, total, 3)
	if len(page2) > 0 {
		assert.NotEqual(t, page1[0].UserID, page2[0].UserID, "翻页结果不应重叠")
		assert.Equal(t, 4, page2[0].Rank, "第二页第一行 rank=4")
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test -tags=integration ./test/integration/... -run TestSpender
```
Expected: 编译失败

- [ ] **Step 3: 实现 spender.go**

`internal/repo/spender.go`:
```go
package repo

import (
	"context"
	"database/sql"
	"fmt"
)

func SpenderTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	if q.Range == RangeAll {
		return spenderAllTime(ctx, db, q)
	}
	return spenderWindowed(ctx, db, q)
}

func spenderAllTime(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)

	countSQL := "SELECT COUNT(*) FROM users u WHERE u.status=1 AND u.deleted_at IS NULL" + hiddenClause
	var total int
	if err := db.QueryRowContext(ctx, countSQL, hiddenArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("spender count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT u.id, %s AS name, u.used_quota
FROM users u
WHERE u.status=1 AND u.deleted_at IS NULL%s
ORDER BY u.used_quota DESC, u.id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, hiddenClause)

	args := append(hiddenArgs, q.PageSize, q.Offset())
	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("spender query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var quota int64
		if err := rows.Scan(&it.UserID, &it.Name, &quota); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(quota)
		it.ValueDisplay = FormatUSD(quota)
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}

func spenderWindowed(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)
	since := RangeStart(q.Range)

	countSQL := fmt.Sprintf(`
SELECT COUNT(DISTINCT l.user_id)
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2 AND l.created_at >= ?
  AND u.status=1 AND u.deleted_at IS NULL%s`, hiddenClause)
	countArgs := append([]any{since}, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("spender count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT l.user_id, %s AS name, SUM(l.quota) AS spent
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2 AND l.created_at >= ?
  AND u.status=1 AND u.deleted_at IS NULL%s
GROUP BY l.user_id, name
ORDER BY spent DESC, l.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, hiddenClause)
	args := append([]any{since}, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("spender query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var spent int64
		if err := rows.Scan(&it.UserID, &it.Name, &spent); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(spent)
		it.ValueDisplay = FormatUSD(spent)
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test -tags=integration ./test/integration/... -run TestSpender -v
```
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/repo/spender.go test/integration/repo_spender_test.go
git commit -m "feat(repo): spender leaderboard (all-time vs windowed)"
```

---

### Task 2.4：充值榜 repo (topup)

**Files:**
- Create: `internal/repo/topup.go`
- Test: `test/integration/repo_topup_test.go`

- [ ] **Step 1: 写测试**

`test/integration/repo_topup_test.go`:
```go
//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
)

func TestTopup_AllTime(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, total, err := repo.TopupTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total, "只有 user 2/3/4 有成功充值")
	require.GreaterOrEqual(t, len(items), 3)
	// user 2 充了 $4000
	assert.Equal(t, int64(2), items[0].UserID)
	assert.InDelta(t, 4000.0, items[0].Value, 0.01)
	// user 3 充了 $3000 (1000 + 2000)
	assert.Equal(t, int64(3), items[1].UserID)
	assert.InDelta(t, 3000.0, items[1].Value, 0.01)
}

func TestTopup_FiltersNonSuccess(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.TopupTop(context.Background(), db, repo.QueryParams{
		Range: repo.RangeAll, Page: 1, PageSize: 100,
	})
	require.NoError(t, err)
	for _, it := range items {
		assert.NotEqual(t, int64(5), it.UserID, "user 5 的充值是 pending，不应入榜")
	}
}

func TestTopup_7d(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.TopupTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range7d, Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	// user 2 (3 天前) 和 user 4 (5 天前) 在 7d 窗口内；user 3 是 10/20 天前不应在内
	for _, it := range items {
		assert.NotEqual(t, int64(3), it.UserID, "user 3 的充值都超过 7 天")
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test -tags=integration ./test/integration/... -run TestTopup
```
Expected: 编译失败

- [ ] **Step 3: 实现 topup.go**

`internal/repo/topup.go`:
```go
package repo

import (
	"context"
	"database/sql"
	"fmt"
)

func TopupTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)

	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND t.create_time >= ?"
		timeArgs = []any{RangeStart(q.Range)}
	}

	countSQL := fmt.Sprintf(`
SELECT COUNT(DISTINCT t.user_id)
FROM top_ups t JOIN users u ON t.user_id=u.id
WHERE t.status='success'%s
  AND u.status=1 AND u.deleted_at IS NULL%s`, timeClause, hiddenClause)
	countArgs := append(timeArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("topup count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT t.user_id, %s AS name, SUM(t.money) AS total_money
FROM top_ups t JOIN users u ON t.user_id=u.id
WHERE t.status='success'%s
  AND u.status=1 AND u.deleted_at IS NULL%s
GROUP BY t.user_id, name
ORDER BY total_money DESC, t.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, timeClause, hiddenClause)

	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("topup query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var money float64
		if err := rows.Scan(&it.UserID, &it.Name, &money); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = money
		it.ValueDisplay = FormatMoney(money)
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test -tags=integration ./test/integration/... -run TestTopup -v
```
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/repo/topup.go test/integration/repo_topup_test.go
git commit -m "feat(repo): topup leaderboard from top_ups (success only)"
```

---

### Task 2.5：吃货榜 repo (foodie)

**Files:**
- Create: `internal/repo/foodie.go`
- Test: `test/integration/repo_foodie_test.go`

- [ ] **Step 1: 写测试**

`test/integration/repo_foodie_test.go`:
```go
//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
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
	// user 3 (15+ 条 claude) 应是榜首
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
	// today 范围内的 log 应该相对少
	if len(items) > 0 {
		assert.GreaterOrEqual(t, items[0].Value, 1.0)
	}
}

func TestFoodie_ValueDisplayIsCount(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, _ := repo.FoodieTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range7d, Page: 1, PageSize: 1,
	})
	require.NotEmpty(t, items)
	assert.Contains(t, items[0].ValueDisplay, ",", "调用次数 > 1000 时含千分位")
	// 或者无千分位（数字小时）
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test -tags=integration ./test/integration/... -run TestFoodie
```
Expected: 编译失败

- [ ] **Step 3: 实现 foodie.go**

`internal/repo/foodie.go`:
```go
package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// FoodieTop 调用次数榜（logs COUNT(*) where type=2）
func FoodieTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)
	since := RangeStart(q.Range)

	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND l.created_at >= ?"
		timeArgs = []any{since}
	}

	countSQL := fmt.Sprintf(`
SELECT COUNT(DISTINCT l.user_id)
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s
  AND u.status=1 AND u.deleted_at IS NULL%s`, timeClause, hiddenClause)
	countArgs := append(timeArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("foodie count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT l.user_id, %s AS name, COUNT(*) AS cnt
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s
  AND u.status=1 AND u.deleted_at IS NULL%s
GROUP BY l.user_id, name
ORDER BY cnt DESC, l.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, timeClause, hiddenClause)

	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("foodie query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var cnt int64
		if err := rows.Scan(&it.UserID, &it.Name, &cnt); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(cnt)
		it.ValueDisplay = FormatCount(cnt) + " 次"
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test -tags=integration ./test/integration/... -run TestFoodie -v
```
Expected: 全部 PASS

- [ ] **Step 5: 提交**

```bash
git add internal/repo/foodie.go test/integration/repo_foodie_test.go
git commit -m "feat(repo): foodie leaderboard by logs count"
```

---

### Task 2.6：死忠粉榜 repo (loyal)

**Files:**
- Create: `internal/repo/loyal.go`
- Test: `test/integration/repo_loyal_test.go`

- [ ] **Step 1: 写测试**

`test/integration/repo_loyal_test.go`:
```go
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
		HiddenUserIDs:  []int64{1},
		LoyalThreshold: 0.8,
		LoyalMinCalls:  10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, items)
	// user 3：15 条 claude + 1 条 gemini，占比 ~94%
	var user3Found bool
	for _, it := range items {
		if it.UserID == 3 {
			user3Found = true
			assert.GreaterOrEqual(t, it.Value, 0.8, "用户 3 占比应 >= 0.8")
			extra, ok := it.Extra.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "claude-sonnet-4", extra["model"])
		}
	}
	assert.True(t, user3Found, "克吹本吹应该上榜")
}

func TestLoyal_TwinFanReturnsGemini(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.LoyalTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 10,
		LoyalThreshold: 0.8,
		LoyalMinCalls:  10,
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
	assert.True(t, user9Found, "双子奶妈应该上榜")
}

func TestLoyal_FiltersSmallSample(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.LoyalTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 100,
		LoyalThreshold: 0.8,
		LoyalMinCalls:  10,
	})
	require.NoError(t, err)
	// user 8 只有 1 条 log，不满足 min_calls=10，不应上榜
	for _, it := range items {
		assert.NotEqual(t, int64(8), it.UserID, "user 8 总调用 < 10，不应上榜")
	}
}

func TestLoyal_HighThresholdFiltersOut(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.LoyalTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 100,
		LoyalThreshold: 0.99, // 几乎不可能
		LoyalMinCalls:  10,
	})
	require.NoError(t, err)
	// user 3 占比 ~94% < 99%，应被过滤
	for _, it := range items {
		assert.NotEqual(t, int64(3), it.UserID)
	}
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test -tags=integration ./test/integration/... -run TestLoyal
```
Expected: 编译失败

- [ ] **Step 3: 实现 loyal.go**

> 注：spec §4.2 用 `GROUP_CONCAT`；这里改用更稳的两步法：
> 1) 子查询统计每个用户的总调用数 + 最大模型调用数 + 占比 + Top 模型名
> 2) 外层按占比过滤 + 排序

`internal/repo/loyal.go`:
```go
package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// LoyalTop 死忠粉：单模型调用占比 >= LoyalThreshold 且总调用 >= LoyalMinCalls
func LoyalTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)
	since := RangeStart(q.Range)

	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND created_at >= ?"
		timeArgs = []any{since}
	}

	// 子查询：每个 (user_id, model_name) 的调用次数；
	// 然后聚合到 user_id 维度，拿 top model + 占比
	// 用 ROW_NUMBER() 选每个用户的 top model（MySQL 8 支持）
	base := fmt.Sprintf(`
WITH per_model AS (
  SELECT user_id, model_name, COUNT(*) AS cnt
  FROM logs WHERE type=2%s
  GROUP BY user_id, model_name
),
ranked AS (
  SELECT user_id, model_name, cnt,
         SUM(cnt) OVER (PARTITION BY user_id) AS total_cnt,
         ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY cnt DESC) AS rn
  FROM per_model
),
agg AS (
  SELECT r.user_id, r.model_name AS top_model, r.cnt AS top_cnt, r.total_cnt,
         r.cnt * 1.0 / r.total_cnt AS ratio
  FROM ranked r WHERE r.rn = 1
)
`, timeClause)

	countSQL := base + fmt.Sprintf(`
SELECT COUNT(*) FROM agg
JOIN users u ON agg.user_id=u.id
WHERE agg.ratio >= ? AND agg.total_cnt >= ?
  AND u.status=1 AND u.deleted_at IS NULL%s`, hiddenClause)
	countArgs := append([]any{}, timeArgs...)
	countArgs = append(countArgs, q.LoyalThreshold, q.LoyalMinCalls)
	countArgs = append(countArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("loyal count: %w", err)
	}

	listSQL := base + fmt.Sprintf(`
SELECT agg.user_id, %s AS name, agg.ratio, agg.top_model, agg.total_cnt
FROM agg JOIN users u ON agg.user_id=u.id
WHERE agg.ratio >= ? AND agg.total_cnt >= ?
  AND u.status=1 AND u.deleted_at IS NULL%s
ORDER BY agg.ratio DESC, agg.total_cnt DESC, agg.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, hiddenClause)

	args := append([]any{}, timeArgs...)
	args = append(args, q.LoyalThreshold, q.LoyalMinCalls)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("loyal query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var ratio float64
		var topModel string
		var totalCnt int64
		if err := rows.Scan(&it.UserID, &it.Name, &ratio, &topModel, &totalCnt); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = ratio
		it.ValueDisplay = fmt.Sprintf("%.1f%%", ratio*100)
		it.Extra = map[string]any{
			"model":       topModel,
			"total_calls": totalCnt,
		}
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test -tags=integration ./test/integration/... -run TestLoyal -v
```
Expected: 全部 PASS

> 如果 MySQL 5.7 不支持窗口函数，需要降级到 GROUP_CONCAT 方案。本项目假设 MySQL 8.0+（spec §4.2 实施提示）。

- [ ] **Step 5: 提交**

```bash
git add internal/repo/loyal.go test/integration/repo_loyal_test.go
git commit -m "feat(repo): loyal fan leaderboard (single-model ratio)"
```

---

### Task 2.7：美食家榜 repo (gourmet)

**Files:**
- Create: `internal/repo/gourmet.go`
- Test: `test/integration/repo_gourmet_test.go`

- [ ] **Step 1: 写测试**

`test/integration/repo_gourmet_test.go`:
```go
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
	// user 7 调用了 7 种不同模型，应该是榜首
	assert.Equal(t, int64(7), items[0].UserID)
	assert.EqualValues(t, 7.0, items[0].Value)
	assert.Equal(t, "7 种", items[0].ValueDisplay)
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test -tags=integration ./test/integration/... -run TestGourmet
```
Expected: 编译失败

- [ ] **Step 3: 实现 gourmet.go**

`internal/repo/gourmet.go`:
```go
package repo

import (
	"context"
	"database/sql"
	"fmt"
)

func GourmetTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)

	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND l.created_at >= ?"
		timeArgs = []any{RangeStart(q.Range)}
	}

	countSQL := fmt.Sprintf(`
SELECT COUNT(DISTINCT l.user_id)
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s
  AND u.status=1 AND u.deleted_at IS NULL%s`, timeClause, hiddenClause)
	countArgs := append(timeArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gourmet count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT l.user_id, %s AS name, COUNT(DISTINCT l.model_name) AS variety
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s
  AND u.status=1 AND u.deleted_at IS NULL%s
GROUP BY l.user_id, name
ORDER BY variety DESC, l.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, timeClause, hiddenClause)

	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("gourmet query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var variety int64
		if err := rows.Scan(&it.UserID, &it.Name, &variety); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(variety)
		it.ValueDisplay = fmt.Sprintf("%d 种", variety)
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test -tags=integration ./test/integration/... -run TestGourmet -v
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/repo/gourmet.go test/integration/repo_gourmet_test.go
git commit -m "feat(repo): gourmet leaderboard (distinct model count)"
```

---

### Task 2.8：单笔王 repo (biteking)

**Files:**
- Create: `internal/repo/biteking.go`
- Test: `test/integration/repo_biteking_test.go`

- [ ] **Step 1: 写测试**

`test/integration/repo_biteking_test.go`:
```go
//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
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
	// user 6 单笔 5000000，应该榜首
	assert.Equal(t, int64(6), items[0].UserID)
	assert.EqualValues(t, 5000000.0, items[0].Value)
	extra, ok := items[0].Extra.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "claude-opus-4", extra["model"])
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test -tags=integration ./test/integration/... -run TestBiteking
```
Expected: 编译失败

- [ ] **Step 3: 实现 biteking.go**

`internal/repo/biteking.go`:
```go
package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// BitekingTop 每个用户的单次最大 quota 消耗
func BitekingTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)

	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND l.created_at >= ?"
		timeArgs = []any{RangeStart(q.Range)}
	}

	// 用窗口函数取每个用户最大 quota 那一行的模型名
	base := fmt.Sprintf(`
WITH max_per_user AS (
  SELECT user_id, model_name, quota,
         ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY quota DESC, id ASC) AS rn
  FROM logs l
  WHERE l.type=2%s
)
`, timeClause)

	countSQL := base + fmt.Sprintf(`
SELECT COUNT(*) FROM max_per_user m
JOIN users u ON m.user_id=u.id
WHERE m.rn=1
  AND u.status=1 AND u.deleted_at IS NULL%s`, hiddenClause)
	countArgs := append([]any{}, timeArgs...)
	countArgs = append(countArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("biteking count: %w", err)
	}

	listSQL := base + fmt.Sprintf(`
SELECT m.user_id, %s AS name, m.quota, m.model_name
FROM max_per_user m JOIN users u ON m.user_id=u.id
WHERE m.rn=1
  AND u.status=1 AND u.deleted_at IS NULL%s
ORDER BY m.quota DESC, m.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, hiddenClause)

	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("biteking query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var quota int64
		var model string
		if err := rows.Scan(&it.UserID, &it.Name, &quota, &model); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(quota)
		it.ValueDisplay = FormatUSD(quota)
		it.Extra = map[string]any{"model": model}
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test -tags=integration ./test/integration/... -run TestBiteking -v
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/repo/biteking.go test/integration/repo_biteking_test.go
git commit -m "feat(repo): biteking leaderboard (max single quota)"
```

---

### Task 2.9：吞噬榜 repo (tokens)

**Files:**
- Create: `internal/repo/tokens.go`
- Test: `test/integration/repo_tokens_test.go`

- [ ] **Step 1: 写测试**

`test/integration/repo_tokens_test.go`:
```go
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
	// user 6 的单笔 10000+20000=30000 token，加上另一条 300+800=1100
	// 但 user 3 是 15 条 (500+1500)*15 = 30000，再加 user 3 的 gemini 200+800=1000
	// 实际上 user 6 = 31100，user 3 = 31000，user 6 略高
	// 不太敏感的断言：top 3 包含 user 6 和 user 3
	ids := map[int64]bool{}
	for i := 0; i < 3 && i < len(items); i++ {
		ids[items[i].UserID] = true
	}
	assert.True(t, ids[6] || ids[3], "user 3 或 6 应在 top3")
}

func TestTokens_ValueDisplayFormatsLargeNumbers(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, _ := repo.TokensTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 1,
	})
	require.NotEmpty(t, items)
	assert.Contains(t, items[0].ValueDisplay, ",", "大数字应含千分位")
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test -tags=integration ./test/integration/... -run TestTokens
```
Expected: 编译失败

- [ ] **Step 3: 实现 tokens.go**

`internal/repo/tokens.go`:
```go
package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// TokensTop 累计 prompt + completion tokens
func TokensTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)

	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND l.created_at >= ?"
		timeArgs = []any{RangeStart(q.Range)}
	}

	countSQL := fmt.Sprintf(`
SELECT COUNT(DISTINCT l.user_id)
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s
  AND u.status=1 AND u.deleted_at IS NULL%s`, timeClause, hiddenClause)
	countArgs := append(timeArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("tokens count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT l.user_id, %s AS name, SUM(l.prompt_tokens + l.completion_tokens) AS total_tokens
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s
  AND u.status=1 AND u.deleted_at IS NULL%s
GROUP BY l.user_id, name
ORDER BY total_tokens DESC, l.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, timeClause, hiddenClause)

	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("tokens query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var tokens int64
		if err := rows.Scan(&it.UserID, &it.Name, &tokens); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(tokens)
		it.ValueDisplay = FormatCount(tokens) + " tok"
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test -tags=integration ./test/integration/... -run TestTokens -v
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/repo/tokens.go test/integration/repo_tokens_test.go
git commit -m "feat(repo): tokens leaderboard (sum prompt+completion)"
```

---

### Task 2.10：熬夜冠军 repo (nightowl)

**Files:**
- Create: `internal/repo/nightowl.go`
- Test: `test/integration/repo_nightowl_test.go`

- [ ] **Step 1: 写测试**

`test/integration/repo_nightowl_test.go`:
```go
//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
)

func TestNightowl_30d(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	items, _, err := repo.NightowlTop(context.Background(), db, repo.QueryParams{
		Range: repo.Range30d, Page: 1, PageSize: 10,
		HiddenUserIDs: []int64{1},
	})
	require.NoError(t, err)
	// user 5 有 5 条凌晨调用记录
	require.NotEmpty(t, items)
	assert.Equal(t, int64(5), items[0].UserID)
	assert.GreaterOrEqual(t, items[0].Value, 5.0)
	assert.Contains(t, items[0].ValueDisplay, "次")
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test -tags=integration ./test/integration/... -run TestNightowl
```
Expected: 编译失败

- [ ] **Step 3: 实现 nightowl.go**

`internal/repo/nightowl.go`:
```go
package repo

import (
	"context"
	"database/sql"
	"fmt"
)

// NightowlTop 熬夜冠军：在 +08:00 时区 0-5 点的调用次数
func NightowlTop(ctx context.Context, db *sql.DB, q QueryParams) ([]LeaderboardItem, int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(q.HiddenUserIDs)

	timeClause := ""
	timeArgs := []any{}
	if q.Range != RangeAll {
		timeClause = " AND l.created_at >= ?"
		timeArgs = []any{RangeStart(q.Range)}
	}

	// HOUR(CONVERT_TZ(FROM_UNIXTIME(...), '+00:00', '+08:00')) BETWEEN 0 AND 5
	// 显式用 '+00:00' 假设 FROM_UNIXTIME 返回 UTC（MySQL 默认行为）。比依赖
	// @@session.time_zone 稳：避免不同部署环境 session timezone 差异导致结果错乱
	hourExpr := "HOUR(CONVERT_TZ(FROM_UNIXTIME(l.created_at), '+00:00', '+08:00'))"

	countSQL := fmt.Sprintf(`
SELECT COUNT(DISTINCT l.user_id)
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s AND %s BETWEEN 0 AND 5
  AND u.status=1 AND u.deleted_at IS NULL%s`, timeClause, hourExpr, hiddenClause)
	countArgs := append(timeArgs, hiddenArgs...)
	var total int
	if err := db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("nightowl count: %w", err)
	}

	listSQL := fmt.Sprintf(`
SELECT l.user_id, %s AS name, COUNT(*) AS cnt
FROM logs l JOIN users u ON l.user_id=u.id
WHERE l.type=2%s AND %s BETWEEN 0 AND 5
  AND u.status=1 AND u.deleted_at IS NULL%s
GROUP BY l.user_id, name
ORDER BY cnt DESC, l.user_id ASC
LIMIT ? OFFSET ?`, DisplayNameSQL, timeClause, hourExpr, hiddenClause)

	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, q.PageSize, q.Offset())

	rows, err := db.QueryContext(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("nightowl query: %w", err)
	}
	defer rows.Close()

	items := []LeaderboardItem{}
	rank := q.Offset() + 1
	for rows.Next() {
		var it LeaderboardItem
		var cnt int64
		if err := rows.Scan(&it.UserID, &it.Name, &cnt); err != nil {
			return nil, 0, err
		}
		it.Rank = rank
		it.Value = float64(cnt)
		it.ValueDisplay = FormatCount(cnt) + " 次"
		items = append(items, it)
		rank++
	}
	return items, total, rows.Err()
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test -tags=integration ./test/integration/... -run TestNightowl -v
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/repo/nightowl.go test/integration/repo_nightowl_test.go
git commit -m "feat(repo): nightowl leaderboard (UTC+8 00:00-05:59 calls)"
```

---

### Task 2.11：个人查询 repo (rank)

**Files:**
- Create: `internal/repo/rank.go`
- Test: `test/integration/repo_rank_test.go`

- [ ] **Step 1: 写测试**

`test/integration/repo_rank_test.go`:
```go
//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/repo"
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
	assert.Equal(t, "plain", users[0].Name, "无 display_name 时回退")
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
	assert.Empty(t, users, "admin 被隐藏后搜不到")
}

func TestRichRankOf(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	rank, err := repo.RichRankOf(context.Background(), db, 2, []int64{1})
	require.NoError(t, err)
	assert.Equal(t, 1, rank, "隐藏 admin 后 user 2 是土豪榜第一")
}

func TestRichRankOf_NotEligible(t *testing.T) {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	rank, err := repo.RichRankOf(context.Background(), db, 10, nil) // 禁用用户
	require.NoError(t, err)
	assert.Equal(t, 0, rank, "被禁用用户返回 0 表示未上榜")
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test -tags=integration ./test/integration/... -run "TestSearchUsers|TestRichRankOf"
```
Expected: 编译失败

- [ ] **Step 3: 实现 rank.go**

`internal/repo/rank.go`:
```go
package repo

import (
	"context"
	"database/sql"
	"fmt"
)

type UserHit struct {
	ID   int64  `json:"user_id"`
	Name string `json:"name"`
}

// SearchUsers 在 users 表里 LIKE 匹配 display_name 或 username，
// 返回候选列表（用 user_id 作为后续 RankOf 调用的主键）
func SearchUsers(ctx context.Context, db *sql.DB, keyword string, hidden []int64, limit int) ([]UserHit, error) {
	if keyword == "" {
		return nil, nil
	}
	hiddenClause, hiddenArgs := BuildHiddenClause(hidden)
	q := fmt.Sprintf(`
SELECT u.id, %s AS name
FROM users u
WHERE u.status=1 AND u.deleted_at IS NULL
  AND (u.display_name LIKE ? OR u.username LIKE ?)%s
ORDER BY u.id ASC
LIMIT ?`, DisplayNameSQL, hiddenClause)

	args := []any{"%" + keyword + "%", "%" + keyword + "%"}
	args = append(args, hiddenArgs...)
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()
	var out []UserHit
	for rows.Next() {
		var u UserHit
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// RichRankOf 返回该用户在土豪榜的排名（1-based）；未上榜返回 0
func RichRankOf(ctx context.Context, db *sql.DB, userID int64, hidden []int64) (int, error) {
	return rankByUserField(ctx, db, userID, hidden, "quota")
}

// SpenderRankOf（按累计 used_quota）
func SpenderRankOf(ctx context.Context, db *sql.DB, userID int64, hidden []int64) (int, error) {
	return rankByUserField(ctx, db, userID, hidden, "used_quota")
}

func rankByUserField(ctx context.Context, db *sql.DB, userID int64, hidden []int64, field string) (int, error) {
	hiddenClause, hiddenArgs := BuildHiddenClause(hidden)

	// 先检查目标用户是否符合上榜条件
	var myVal int64
	row := db.QueryRowContext(ctx,
		"SELECT "+field+" FROM users WHERE id=? AND status=1 AND deleted_at IS NULL",
		userID)
	if err := row.Scan(&myVal); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	// 检查是否在 hidden 列表
	for _, h := range hidden {
		if h == userID {
			return 0, nil
		}
	}

	q := fmt.Sprintf(`
SELECT COUNT(*) + 1 FROM users u
WHERE u.status=1 AND u.deleted_at IS NULL
  AND u.%s > ?%s`, field, hiddenClause)
	args := []any{myVal}
	args = append(args, hiddenArgs...)
	var rank int
	if err := db.QueryRowContext(ctx, q, args...).Scan(&rank); err != nil {
		return 0, err
	}
	return rank, nil
}

// LogAggRankOf 适用于 foodie/tokens/gourmet/biteking/nightowl 等基于 logs 聚合的榜单
// metric: "count"|"distinct_model"|"max_quota"|"sum_tokens"|"night_count"
func LogAggRankOf(ctx context.Context, db *sql.DB, userID int64, r Range, metric string, hidden []int64) (int, error) {
	// 该用户的聚合值
	myVal, ok, err := logAggValue(ctx, db, userID, r, metric)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	for _, h := range hidden {
		if h == userID {
			return 0, nil
		}
	}
	hiddenClause, hiddenArgs := BuildHiddenClause(hidden)

	timeClause := ""
	timeArgs := []any{}
	if r != RangeAll {
		timeClause = " AND l.created_at >= ?"
		timeArgs = []any{RangeStart(r)}
	}
	aggExpr, extraWhere := metricExpr(metric)

	q := fmt.Sprintf(`
SELECT COUNT(*) + 1 FROM (
  SELECT %s AS v FROM logs l JOIN users u ON l.user_id=u.id
  WHERE l.type=2%s%s
    AND u.status=1 AND u.deleted_at IS NULL%s
  GROUP BY l.user_id
  HAVING v > ?
) t`, aggExpr, timeClause, extraWhere, hiddenClause)
	args := append([]any{}, timeArgs...)
	args = append(args, hiddenArgs...)
	args = append(args, myVal)

	var rank int
	if err := db.QueryRowContext(ctx, q, args...).Scan(&rank); err != nil {
		return 0, err
	}
	return rank, nil
}

func metricExpr(metric string) (aggExpr, extraWhere string) {
	switch metric {
	case "count":
		return "COUNT(*)", ""
	case "distinct_model":
		return "COUNT(DISTINCT l.model_name)", ""
	case "max_quota":
		return "MAX(l.quota)", ""
	case "sum_tokens":
		return "SUM(l.prompt_tokens + l.completion_tokens)", ""
	case "night_count":
		return "COUNT(*)", " AND HOUR(CONVERT_TZ(FROM_UNIXTIME(l.created_at), '+00:00', '+08:00')) BETWEEN 0 AND 5"
	}
	return "COUNT(*)", ""
}

func logAggValue(ctx context.Context, db *sql.DB, userID int64, r Range, metric string) (float64, bool, error) {
	timeClause := ""
	timeArgs := []any{}
	if r != RangeAll {
		timeClause = " AND created_at >= ?"
		timeArgs = []any{RangeStart(r)}
	}
	aggExpr, extraWhere := metricExpr(metric)
	q := fmt.Sprintf("SELECT %s FROM logs WHERE type=2 AND user_id=?%s%s", aggExpr, timeClause, extraWhere)
	args := append([]any{userID}, timeArgs...)
	var v sql.NullFloat64
	if err := db.QueryRowContext(ctx, q, args...).Scan(&v); err != nil {
		return 0, false, err
	}
	if !v.Valid || v.Float64 == 0 {
		return 0, false, nil
	}
	return v.Float64, true, nil
}
```

> 注：上面只实现 9 个榜单中通用的 6 类排名函数（rich/spender + 5 个 logs 聚合），充值榜（topup）的 RankOf 和死忠粉（loyal）的 RankOf 较特殊，会在 service 层用专门逻辑（直接调用对应 Top 方法搜索包含 userID 的页）。

- [ ] **Step 4: 跑测试看通过**

```bash
go test -tags=integration ./test/integration/... -run "TestSearchUsers|TestRichRankOf" -v
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/repo/rank.go test/integration/repo_rank_test.go
git commit -m "feat(repo): search users + rank-of queries for 9 leaderboards"
```

---

## Phase 3：Service + Middleware + Handler

### Task 3.1：service 层（统一编排 9 个榜单 + 缓存）

**Files:**
- Create: `internal/service/leaderboard.go`
- Create: `internal/service/meta.go`
- Test: `test/integration/service_test.go`

- [ ] **Step 1: 写 meta（榜单元信息）**

`internal/service/meta.go`:
```go
package service

type LeaderboardMeta struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Emoji       string   `json:"emoji"`
	Description string   `json:"description"`
	Ranges      []string `json:"ranges"`
}

var AllMeta = []LeaderboardMeta{
	{Type: "rich", Name: "土豪榜", Emoji: "💰", Description: "钱包余额最多 · 谁是钱袋子之王", Ranges: []string{"all"}},
	{Type: "spender", Name: "散财榜", Emoji: "💸", Description: "累计消费最多 · 挥金如土", Ranges: []string{"today", "7d", "30d", "all"}},
	{Type: "topup", Name: "充值榜", Emoji: "💎", Description: "充值金额最多", Ranges: []string{"7d", "30d", "all"}},
	{Type: "foodie", Name: "吃货榜", Emoji: "🍴", Description: "调用次数最多 · 吃货之王", Ranges: []string{"today", "7d", "30d"}},
	{Type: "loyal", Name: "死忠粉榜", Emoji: "🎯", Description: "对单一模型最专一", Ranges: []string{"7d", "30d", "all"}},
	{Type: "gourmet", Name: "美食家榜", Emoji: "🌈", Description: "尝过的模型种类最多", Ranges: []string{"7d", "30d", "all"}},
	{Type: "biteking", Name: "单笔王", Emoji: "🔥", Description: "单次最大消耗 · 豪赌一把", Ranges: []string{"7d", "30d", "all"}},
	{Type: "tokens", Name: "吞噬榜", Emoji: "⚡", Description: "累计 token 消耗最多", Ranges: []string{"today", "7d", "30d"}},
	{Type: "nightowl", Name: "熬夜冠军", Emoji: "🌙", Description: "凌晨 0-5 点调用最多", Ranges: []string{"7d", "30d"}},
}

func MetaByType(t string) (LeaderboardMeta, bool) {
	for _, m := range AllMeta {
		if m.Type == t {
			return m, true
		}
	}
	return LeaderboardMeta{}, false
}

func ValidType(t string) bool { _, ok := MetaByType(t); return ok }

func RangeSupported(t, r string) bool {
	m, ok := MetaByType(t)
	if !ok {
		return false
	}
	for _, x := range m.Ranges {
		if x == r {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: 写 service 主体**

`internal/service/leaderboard.go`:
```go
package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yourname/newapi-leaderboard/internal/cache"
	"github.com/yourname/newapi-leaderboard/internal/config"
	"github.com/yourname/newapi-leaderboard/internal/repo"
)

type Service struct {
	db    *sql.DB
	cache *cache.Cache
	cfg   *config.Config
}

func New(db *sql.DB, c *cache.Cache, cfg *config.Config) *Service {
	return &Service{db: db, cache: c, cfg: cfg}
}

type Response struct {
	Type         string                 `json:"type"`
	Range        string                 `json:"range"`
	Total        int                    `json:"total"`
	Page         int                    `json:"page"`
	PageSize     int                    `json:"page_size"`
	List         []repo.LeaderboardItem `json:"list"`
	UpdatedAt    int64                  `json:"updated_at"`
	Cached       bool                   `json:"cached"`
	Stale        bool                   `json:"stale,omitempty"`
}

type fetchResult struct {
	Items []repo.LeaderboardItem
	Total int
	At    int64
}

// GetHiddenUserIDs 合并 env 中的 HIDDEN_USER_IDS 和 admin store 中的临时隐藏
func (s *Service) GetHiddenUserIDs(extra []int64) []int64 {
	out := make([]int64, 0, len(s.cfg.HiddenUserIDs)+len(extra))
	out = append(out, s.cfg.HiddenUserIDs...)
	out = append(out, extra...)
	return out
}

// Get 是榜单查询的统一入口
func (s *Service) Get(ctx context.Context, lbType, rangeStr string, page, pageSize int, extraHidden []int64) (*Response, error) {
	if !ValidType(lbType) {
		return nil, fmt.Errorf("unknown type: %s", lbType)
	}
	if !RangeSupported(lbType, rangeStr) {
		return nil, fmt.Errorf("range %s not supported for %s", rangeStr, lbType)
	}

	cacheKey := fmt.Sprintf("lb:%s:%s:p%d:s%d", lbType, rangeStr, page, pageSize)
	ttl := time.Duration(s.cfg.CacheTTLLogs) * time.Second
	// users 类（实时计算的用户字段）用更短 TTL：
	// rich 总是；spender 仅在累计 (all) 模式下才用 users.used_quota
	if lbType == "rich" || (lbType == "spender" && rangeStr == "all") {
		ttl = time.Duration(s.cfg.CacheTTLUsers) * time.Second
	}

	val, stale, err := s.cache.GetOrLoad(cacheKey, ttl, func() (any, error) {
		items, total, err := s.fetch(ctx, lbType, rangeStr, page, pageSize, extraHidden)
		if err != nil {
			return nil, err
		}
		return fetchResult{Items: items, Total: total, At: time.Now().Unix()}, nil
	})
	if err != nil {
		return nil, err
	}
	fr := val.(fetchResult)
	return &Response{
		Type:      lbType,
		Range:     rangeStr,
		Total:     fr.Total,
		Page:      page,
		PageSize:  pageSize,
		List:      fr.Items,
		UpdatedAt: fr.At,
		Cached:    true,
		Stale:     stale,
	}, nil
}

func (s *Service) fetch(ctx context.Context, t, r string, page, ps int, extraHidden []int64) ([]repo.LeaderboardItem, int, error) {
	q := repo.QueryParams{
		Range:          repo.Range(r),
		Page:           page,
		PageSize:       ps,
		HiddenUserIDs:  s.GetHiddenUserIDs(extraHidden),
		LoyalThreshold: s.cfg.LoyalThreshold,
		LoyalMinCalls:  s.cfg.LoyalMinCalls,
	}
	switch t {
	case "rich":
		return repo.RichTop(ctx, s.db, q)
	case "spender":
		return repo.SpenderTop(ctx, s.db, q)
	case "topup":
		return repo.TopupTop(ctx, s.db, q)
	case "foodie":
		return repo.FoodieTop(ctx, s.db, q)
	case "loyal":
		return repo.LoyalTop(ctx, s.db, q)
	case "gourmet":
		return repo.GourmetTop(ctx, s.db, q)
	case "biteking":
		return repo.BitekingTop(ctx, s.db, q)
	case "tokens":
		return repo.TokensTop(ctx, s.db, q)
	case "nightowl":
		return repo.NightowlTop(ctx, s.db, q)
	}
	return nil, 0, fmt.Errorf("unreachable: %s", t)
}

// SearchAndRank 个人查询：先 LIKE 搜用户，再对每个 user 算 9 榜排名
type PersonalRank struct {
	UserID int64                  `json:"user_id"`
	Name   string                 `json:"name"`
	Ranks  map[string]PersonalEntry `json:"ranks"`
}

type PersonalEntry struct {
	Rank  int     `json:"rank"`
	Value float64 `json:"value"`
}

func (s *Service) SearchAndRank(ctx context.Context, keyword string, extraHidden []int64) ([]PersonalRank, error) {
	hidden := s.GetHiddenUserIDs(extraHidden)
	users, err := repo.SearchUsers(ctx, s.db, keyword, hidden, 10)
	if err != nil {
		return nil, err
	}
	out := make([]PersonalRank, 0, len(users))
	for _, u := range users {
		pr := PersonalRank{UserID: u.ID, Name: u.Name, Ranks: map[string]PersonalEntry{}}
		richRank, _ := repo.RichRankOf(ctx, s.db, u.ID, hidden)
		if richRank > 0 {
			pr.Ranks["rich"] = PersonalEntry{Rank: richRank}
		}
		spenderRank, _ := repo.SpenderRankOf(ctx, s.db, u.ID, hidden)
		if spenderRank > 0 {
			pr.Ranks["spender"] = PersonalEntry{Rank: spenderRank}
		}
		for _, m := range []struct{ t, metric, r string }{
			{"foodie", "count", "7d"},
			{"gourmet", "distinct_model", "30d"},
			{"biteking", "max_quota", "30d"},
			{"tokens", "sum_tokens", "7d"},
			{"nightowl", "night_count", "30d"},
		} {
			rk, _ := repo.LogAggRankOf(ctx, s.db, u.ID, repo.Range(m.r), m.metric, hidden)
			if rk > 0 {
				pr.Ranks[m.t] = PersonalEntry{Rank: rk}
			}
		}
		out = append(out, pr)
	}
	return out, nil
}
```

- [ ] **Step 3: 写集成测试**

`test/integration/service_test.go`:
```go
//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/cache"
	"github.com/yourname/newapi-leaderboard/internal/config"
	"github.com/yourname/newapi-leaderboard/internal/service"
)

func newTestService(t *testing.T) *service.Service {
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	c := cache.New()
	cfg := &config.Config{
		CacheTTLUsers:  60,
		CacheTTLLogs:   300,
		HiddenUserIDs:  []int64{1},
		LoyalThreshold: 0.8,
		LoyalMinCalls:  10,
	}
	return service.New(db, c, cfg)
}

func TestService_GetRich(t *testing.T) {
	s := newTestService(t)
	resp, err := s.Get(context.Background(), "rich", "all", 1, 5, nil)
	require.NoError(t, err)
	assert.Equal(t, "rich", resp.Type)
	require.NotEmpty(t, resp.List)
	assert.Equal(t, int64(2), resp.List[0].UserID, "隐藏 admin 后 user 2 是第一")
}

func TestService_InvalidRange(t *testing.T) {
	s := newTestService(t)
	_, err := s.Get(context.Background(), "rich", "yesterday", 1, 5, nil)
	require.Error(t, err)
}

func TestService_RangeNotSupported(t *testing.T) {
	s := newTestService(t)
	_, err := s.Get(context.Background(), "rich", "7d", 1, 5, nil) // rich 只支持 all
	require.Error(t, err)
}

func TestService_SearchAndRank(t *testing.T) {
	s := newTestService(t)
	results, err := s.SearchAndRank(context.Background(), "克吹", nil)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	r := results[0]
	assert.Equal(t, int64(3), r.UserID)
	assert.Contains(t, r.Ranks, "rich")
	assert.Contains(t, r.Ranks, "foodie")
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test -tags=integration ./test/integration/... -run TestService -v
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/service/ test/integration/service_test.go
git commit -m "feat(service): leaderboard orchestration with cache + personal rank"
```

---

### Task 3.2：persist 包（admin.json 持久化）

**Files:**
- Create: `internal/persist/admin_store.go`
- Test: `internal/persist/admin_store_test.go`

- [ ] **Step 1: 写测试**

`internal/persist/admin_store_test.go`:
```go
package persist

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminStore_AddRemoveHidden(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "admin.json")
	store, err := NewAdminStore(path)
	require.NoError(t, err)
	assert.Empty(t, store.HiddenUserIDs())

	require.NoError(t, store.AddHidden(42))
	require.NoError(t, store.AddHidden(99))
	require.NoError(t, store.AddHidden(42)) // 重复
	ids := store.HiddenUserIDs()
	assert.ElementsMatch(t, []int64{42, 99}, ids)

	require.NoError(t, store.RemoveHidden(42))
	assert.Equal(t, []int64{99}, store.HiddenUserIDs())
}

func TestAdminStore_PersistsAcrossReopens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "admin.json")
	s1, _ := NewAdminStore(path)
	_ = s1.AddHidden(7)
	s2, err := NewAdminStore(path)
	require.NoError(t, err)
	assert.Equal(t, []int64{7}, s2.HiddenUserIDs())
}
```

- [ ] **Step 2: 跑测试看失败**

```bash
go test ./internal/persist/...
```
Expected: 编译失败

- [ ] **Step 3: 实现 admin_store.go**

`internal/persist/admin_store.go`:
```go
package persist

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type AdminStore struct {
	mu     sync.RWMutex
	path   string
	state  state
}

type state struct {
	HiddenUserIDs []int64 `json:"hidden_user_ids"`
}

func NewAdminStore(path string) (*AdminStore, error) {
	s := &AdminStore{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *AdminStore) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	return json.Unmarshal(b, &s.state)
}

func (s *AdminStore) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func (s *AdminStore) HiddenUserIDs() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]int64, len(s.state.HiddenUserIDs))
	copy(out, s.state.HiddenUserIDs)
	return out
}

func (s *AdminStore) AddHidden(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.state.HiddenUserIDs {
		if x == id {
			return nil
		}
	}
	s.state.HiddenUserIDs = append(s.state.HiddenUserIDs, id)
	sort.Slice(s.state.HiddenUserIDs, func(i, j int) bool { return s.state.HiddenUserIDs[i] < s.state.HiddenUserIDs[j] })
	return s.save()
}

func (s *AdminStore) RemoveHidden(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.state.HiddenUserIDs[:0]
	for _, x := range s.state.HiddenUserIDs {
		if x != id {
			out = append(out, x)
		}
	}
	s.state.HiddenUserIDs = out
	return s.save()
}
```

- [ ] **Step 4: 跑测试看通过**

```bash
go test ./internal/persist/... -v -race
```
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/persist/
git commit -m "feat(persist): admin.json store for temporary hidden user IDs"
```

---

### Task 3.3：middleware（CORS + recovery + ratelimit + admin auth）

**Files:**
- Create: `internal/middleware/cors.go`
- Create: `internal/middleware/recovery.go`
- Create: `internal/middleware/ratelimit.go`
- Create: `internal/middleware/admin_auth.go`
- Test: `internal/middleware/admin_auth_test.go`

- [ ] **Step 1: 写 cors.go**

`internal/middleware/cors.go`:
```go
package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 全开，配合嵌入式 widget 场景
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		// 不要 Allow-Credentials = true（与 Origin: * 冲突）
		// frame-ancestors 让任意网站嵌入
		c.Header("Content-Security-Policy", "frame-ancestors *")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 2: 写 recovery.go**

`internal/middleware/recovery.go`:
```go
package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC %s %s: %v\n%s", c.Request.Method, c.Request.URL.Path, r, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code": 500, "msg": "服务内部错误",
				})
			}
		}()
		c.Next()
	}
}
```

- [ ] **Step 3: 写 ratelimit.go**

`internal/middleware/ratelimit.go`:
```go
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

func newIPLimiter(perMin int) *ipLimiter {
	return &ipLimiter{
		limiters: map[string]*rate.Limiter{},
		rps:      rate.Limit(float64(perMin) / 60.0),
		burst:    perMin / 4,
	}
}

func (l *ipLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	lim, ok := l.limiters[ip]
	if !ok {
		lim = rate.NewLimiter(l.rps, l.burst)
		l.limiters[ip] = lim
	}
	return lim
}

// 后台清理任务防内存涨爆（每 10 分钟清一次空 limiter）
func (l *ipLimiter) startGC() {
	go func() {
		t := time.NewTicker(10 * time.Minute)
		for range t.C {
			l.mu.Lock()
			for ip, lim := range l.limiters {
				if lim.Allow() && lim.Tokens() >= float64(l.burst-1) {
					delete(l.limiters, ip)
				}
			}
			l.mu.Unlock()
		}
	}()
}

func RateLimit(perMin int) gin.HandlerFunc {
	lim := newIPLimiter(perMin)
	lim.startGC()
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !lim.get(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": 429, "msg": "请求过于频繁，请稍后再试",
			})
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 4: 写 admin_auth.go + 测试**

`internal/middleware/admin_auth.go`:
```go
package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AdminAuth(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expectedToken == "" {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"code": 404, "msg": "admin not configured",
			})
			return
		}
		got := c.GetHeader("Authorization")
		got = strings.TrimPrefix(got, "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(expectedToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401, "msg": "unauthorized",
			})
			return
		}
		c.Next()
	}
}
```

`internal/middleware/admin_auth_test.go`:
```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAdminAuth_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", AdminAuth("secret"), func(c *gin.Context) { c.Status(200) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuth_Wrong(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", AdminAuth("secret"), func(c *gin.Context) { c.Status(200) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuth_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", AdminAuth("secret"), func(c *gin.Context) { c.Status(200) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminAuth_DisabledWhenNoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", AdminAuth(""), func(c *gin.Context) { c.Status(200) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)
}
```

- [ ] **Step 5: 跑测试 + 提交**

```bash
go test ./internal/middleware/... -v -race
git add internal/middleware/
git commit -m "feat(middleware): CORS / recovery / ratelimit / admin auth"
```

---

### Task 3.4：handler（meta / leaderboard / rank / health）

**Files:**
- Create: `internal/handler/meta.go`
- Create: `internal/handler/leaderboard.go`
- Create: `internal/handler/rank.go`
- Create: `internal/handler/health.go`
- Test: `test/integration/handler_test.go`

- [ ] **Step 1: 写 handler 文件**

`internal/handler/meta.go`:
```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yourname/newapi-leaderboard/internal/config"
	"github.com/yourname/newapi-leaderboard/internal/service"
)

type EmbedDefault struct {
	Tabs    []string `json:"tabs"`
	SiteURL string   `json:"site_url"`
	SiteName string  `json:"site_name"`
}

type MetaResponse struct {
	Leaderboards []service.LeaderboardMeta `json:"leaderboards"`
	Embed        EmbedDefault              `json:"embed"`
	Version      string                    `json:"version"`
}

func Meta(cfg *config.Config, version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": MetaResponse{
				Leaderboards: service.AllMeta,
				Embed: EmbedDefault{
					Tabs:     cfg.EmbedTabsDefault,
					SiteURL:  cfg.SiteURL,
					SiteName: cfg.SiteName,
				},
				Version: version,
			},
		})
	}
}
```

`internal/handler/leaderboard.go`:
```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/yourname/newapi-leaderboard/internal/persist"
	"github.com/yourname/newapi-leaderboard/internal/service"
)

func Leaderboard(s *service.Service, store *persist.AdminStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		lbType := c.Param("type")
		rangeStr := c.DefaultQuery("range", "7d")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		var extraHidden []int64
		if store != nil {
			extraHidden = store.HiddenUserIDs()
		}

		resp, err := s.Get(c.Request.Context(), lbType, rangeStr, page, pageSize, extraHidden)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
	}
}
```

`internal/handler/rank.go`:
```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yourname/newapi-leaderboard/internal/persist"
	"github.com/yourname/newapi-leaderboard/internal/service"
)

func Rank(s *service.Service, store *persist.AdminStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyword := c.Param("keyword")
		if len(keyword) == 0 || len(keyword) > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "keyword 长度需 1-50"})
			return
		}
		var extra []int64
		if store != nil {
			extra = store.HiddenUserIDs()
		}
		results, err := s.SearchAndRank(c.Request.Context(), keyword, extra)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
			return
		}
		if len(results) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"data": gin.H{"found": false, "keyword": keyword},
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{"found": true, "keyword": keyword, "results": results},
		})
	}
}
```

`internal/handler/health.go`:
```go
package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yourname/newapi-leaderboard/internal/db"
)

func Health(d *sql.DB, version string) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbOK := db.Health(d) == nil
		status := http.StatusOK
		if !dbOK {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{
			"code": 0,
			"data": gin.H{
				"db":      dbOK,
				"version": version,
			},
		})
	}
}
```

- [ ] **Step 2: 写集成测试**

`test/integration/handler_test.go`:
```go
//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/cache"
	"github.com/yourname/newapi-leaderboard/internal/config"
	"github.com/yourname/newapi-leaderboard/internal/handler"
	"github.com/yourname/newapi-leaderboard/internal/service"
)

func buildRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	db, _ := SharedDB(t)
	ResetSeed(t, db)
	cfg := &config.Config{HiddenUserIDs: []int64{1}, LoyalThreshold: 0.8, LoyalMinCalls: 10,
		CacheTTLUsers: 60, CacheTTLLogs: 300, EmbedTabsDefault: []string{"rich"}}
	c := cache.New()
	svc := service.New(db, c, cfg)
	r := gin.New()
	r.GET("/api/meta", handler.Meta(cfg, "test"))
	r.GET("/api/leaderboard/:type", handler.Leaderboard(svc, nil))
	r.GET("/api/rank/:keyword", handler.Rank(svc, nil))
	r.GET("/api/health", handler.Health(db, "test"))
	return r
}

func TestHandler_Meta(t *testing.T) {
	r := buildRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/meta", nil))
	assert.Equal(t, 200, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.EqualValues(t, 0, resp["code"])
}

func TestHandler_Rich(t *testing.T) {
	r := buildRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/leaderboard/rich?range=all&page=1&page_size=5", nil))
	assert.Equal(t, 200, w.Code)
}

func TestHandler_BadType(t *testing.T) {
	r := buildRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/leaderboard/fake?range=all", nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_RankFound(t *testing.T) {
	r := buildRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/rank/克吹", nil))
	assert.Equal(t, 200, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, true, data["found"])
}

func TestHandler_RankNotFound(t *testing.T) {
	r := buildRouter(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/rank/zzznoone", nil))
	assert.Equal(t, 200, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, false, data["found"])
}
```

- [ ] **Step 3: 跑测试 + 提交**

```bash
go test -tags=integration ./test/integration/... -run TestHandler -v
git add internal/handler/ test/integration/handler_test.go
git commit -m "feat(handler): meta / leaderboard / rank / health endpoints"
```

---

### Task 3.5：admin handler（缓存清理 / 隐藏用户 / 统计）

**Files:**
- Create: `internal/handler/admin.go`
- Test: `test/integration/admin_test.go`

- [ ] **Step 1: 实现**

`internal/handler/admin.go`:
```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/yourname/newapi-leaderboard/internal/cache"
	"github.com/yourname/newapi-leaderboard/internal/config"
	"github.com/yourname/newapi-leaderboard/internal/persist"
)

func AdminClearCache(c *cache.Cache) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		prefix := ctx.Query("prefix")
		n := 0
		if prefix == "" {
			c.Clear()
		} else {
			n = c.DeletePrefix(prefix)
		}
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"cleared": n}})
	}
}

func AdminGetHidden(store *persist.AdminStore, cfg *config.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"env":   cfg.HiddenUserIDs,
				"admin": store.HiddenUserIDs(),
			},
		})
	}
}

func AdminAddHidden(store *persist.AdminStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var body struct {
			UserID int64 `json:"user_id"`
		}
		if err := ctx.ShouldBindJSON(&body); err != nil || body.UserID <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid user_id"})
			return
		}
		if err := store.AddHidden(body.UserID); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"code": 0})
	}
}

func AdminRemoveHidden(store *persist.AdminStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid id"})
			return
		}
		if err := store.RemoveHidden(id); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"code": 0})
	}
}

func AdminStats(c *cache.Cache) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		s := c.Stats()
		hitRate := 0.0
		if total := s.Hits + s.Misses; total > 0 {
			hitRate = float64(s.Hits) / float64(total)
		}
		ctx.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"cache_hits":   s.Hits,
				"cache_misses": s.Misses,
				"cache_size":   s.Size,
				"hit_rate":     hitRate,
			},
		})
	}
}
```

- [ ] **Step 2: 写集成测试**

`test/integration/admin_test.go`:
```go
//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourname/newapi-leaderboard/internal/cache"
	"github.com/yourname/newapi-leaderboard/internal/config"
	"github.com/yourname/newapi-leaderboard/internal/handler"
	"github.com/yourname/newapi-leaderboard/internal/middleware"
	"github.com/yourname/newapi-leaderboard/internal/persist"
)

func buildAdminRouter(t *testing.T, token string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	c := cache.New()
	cfg := &config.Config{HiddenUserIDs: []int64{1}, AdminToken: token}
	store, _ := persist.NewAdminStore(filepath.Join(t.TempDir(), "admin.json"))
	r := gin.New()
	g := r.Group("/admin", middleware.AdminAuth(token))
	g.POST("/cache/clear", handler.AdminClearCache(c))
	g.GET("/hidden-users", handler.AdminGetHidden(store, cfg))
	g.POST("/hidden-users", handler.AdminAddHidden(store))
	g.DELETE("/hidden-users/:id", handler.AdminRemoveHidden(store))
	g.GET("/stats", handler.AdminStats(c))
	return r
}

func TestAdmin_AuthRequired(t *testing.T) {
	r := buildAdminRouter(t, "secret")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/admin/cache/clear", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdmin_AddAndRemoveHidden(t *testing.T) {
	r := buildAdminRouter(t, "secret")
	// 添加
	req := httptest.NewRequest("POST", "/admin/hidden-users",
		bytes.NewBufferString(`{"user_id":99}`))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	// 列出
	req2 := httptest.NewRequest("GET", "/admin/hidden-users", nil)
	req2.Header.Set("Authorization", "Bearer secret")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	admin := data["admin"].([]any)
	assert.EqualValues(t, 99, admin[0])
	// 删除
	req3 := httptest.NewRequest("DELETE", "/admin/hidden-users/99", nil)
	req3.Header.Set("Authorization", "Bearer secret")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, 200, w3.Code)
}
```

- [ ] **Step 3: 跑测试 + 提交**

```bash
go test -tags=integration ./test/integration/... -run TestAdmin -v
git add internal/handler/admin.go test/integration/admin_test.go
git commit -m "feat(handler): admin endpoints (cache / hidden / stats)"
```

---

## Phase 4：静态资源分发 + main.go 装配 + 端到端冒烟

> **关于 go:embed 路径的约定（重要）**：`//go:embed` 不支持 `..` 路径，所以前端 dist 目录必须与 embed 指令所在的 .go 文件同级。本项目约定：
> - 前端构建产物 `web/dist/*` 由 `web/scripts/copy-to-embed.mjs` 拷贝到 `internal/embed/dist/`（Task 5.1 的 build 脚本会自动做）
> - `internal/embed/dist.go` 用 `//go:embed all:dist` 包住这个目录
> - 仓库始终保留 `internal/embed/dist/.gitkeep` 和 3 个占位 HTML，让 embed 编译不报错；CI / 本地构建后会被真实产物覆盖

### Task 4.1：embed 静态资源 + 路由分发

**Files:**
- Create: `internal/embed/dist.go`
- Create: `internal/embed/static.go`
- Create: `internal/embed/dist/.gitkeep`
- Create: `internal/embed/dist/index.html`（占位，Phase 5 由 Vite 覆盖）
- Create: `internal/embed/dist/embed.html`（占位）
- Create: `internal/embed/dist/admin.html`（占位）
- Modify: `.gitignore`

- [ ] **Step 1: 写最小占位 dist 文件**

`internal/embed/dist/index.html`:
```html
<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>NewAPI 排行榜</title></head>
<body><div id="root">前端未构建。请先在 web/ 目录下执行 pnpm build</div></body></html>
```
`internal/embed/dist/embed.html`、`internal/embed/dist/admin.html` 内容相同（仅 title 不同）。

- [ ] **Step 2: 写 `dist.go`（持有 embed 指令）**

`internal/embed/dist.go`:
```go
package embed

import "embed"

//go:embed all:dist
var distFS embed.FS
```

- [ ] **Step 3: 写 `static.go`**

`internal/embed/static.go`:
```go
package embed

import (
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Register 把 SPA 三个入口和静态资源挂到 Gin 路由
func Register(r *gin.Engine) {
	root, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	r.GET("/", serveFile(root, "index.html", "text/html; charset=utf-8"))
	r.GET("/embed", serveFile(root, "embed.html", "text/html; charset=utf-8"))
	r.GET("/admin", serveFile(root, "admin.html", "text/html; charset=utf-8"))

	assetsFS, _ := fs.Sub(root, "assets")
	assetsHandler := http.FileServer(http.FS(assetsFS))
	r.GET("/assets/*filepath", func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.StripPrefix("/assets", assetsHandler).ServeHTTP(c.Writer, c.Request)
	})

	r.GET("/favicon.ico", func(c *gin.Context) {
		if data, err := fs.ReadFile(root, "favicon.ico"); err == nil {
			c.Data(200, "image/x-icon", data)
			return
		}
		c.Status(204)
	})
}

func serveFile(root fs.FS, name, ct string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := fs.ReadFile(root, name)
		if err != nil {
			c.String(404, "not found: %s", name)
			return
		}
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Data(200, ct, data)
	}
}
```

- [ ] **Step 4: 更新 `.gitignore`**

`.gitignore` 末尾追加：
```
# 前端构建产物（CI/本地 build 时由 copy-to-embed.mjs 写入）
internal/embed/dist/assets/
# 但保留占位 HTML 和 .gitkeep
!internal/embed/dist/index.html
!internal/embed/dist/embed.html
!internal/embed/dist/admin.html
!internal/embed/dist/.gitkeep
```

- [ ] **Step 5: 编译验证**

```bash
go build ./...
```
Expected: PASS（占位 HTML 让 embed 不报 "no matching files"）

- [ ] **Step 6: 提交**

```bash
git add internal/embed/ .gitignore
git commit -m "feat(embed): static asset router with go:embed (3 SPA entries + assets)"
```

---

### Task 4.2：cmd/leaderboard/main.go 装配

**Files:**
- Create: `cmd/leaderboard/main.go`

- [ ] **Step 1: 写 main.go**

`cmd/leaderboard/main.go`:
```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yourname/newapi-leaderboard/internal/cache"
	"github.com/yourname/newapi-leaderboard/internal/config"
	"github.com/yourname/newapi-leaderboard/internal/db"
	embedpkg "github.com/yourname/newapi-leaderboard/internal/embed"
	"github.com/yourname/newapi-leaderboard/internal/handler"
	"github.com/yourname/newapi-leaderboard/internal/middleware"
	"github.com/yourname/newapi-leaderboard/internal/persist"
	"github.com/yourname/newapi-leaderboard/internal/service"
)

var version = "dev"  // 由 -ldflags="-X main.version=..." 注入

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// 1) DB
	database, err := db.OpenPool(cfg)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()
	if err := db.CheckSchema(database); err != nil {
		log.Fatalf("schema check: %v", err)
	}
	if _, err := database.Exec("SET SESSION group_concat_max_len = 65535"); err != nil {
		log.Printf("warn: SET group_concat_max_len: %v", err)
	}

	// 2) Cache
	memCache := cache.New()

	// 3) Persist
	dataDir := "data"
	if envDir := os.Getenv("DATA_DIR"); envDir != "" {
		dataDir = envDir
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	store, err := persist.NewAdminStore(filepath.Join(dataDir, "admin.json"))
	if err != nil {
		log.Fatalf("admin store: %v", err)
	}

	// 4) Service
	svc := service.New(database, memCache, cfg)

	// 5) Router
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.Recovery(), middleware.CORS())

	// 公开 API（限流）
	api := r.Group("/api", middleware.RateLimit(cfg.RateLimitPerMin))
	api.GET("/meta", handler.Meta(cfg, version))
	api.GET("/leaderboard/:type", handler.Leaderboard(svc, store))
	api.GET("/rank/:keyword", handler.Rank(svc, store))
	api.GET("/health", handler.Health(database, version))

	// 管理 API
	if cfg.AdminEnabled() {
		admin := r.Group("/admin", middleware.AdminAuth(cfg.AdminToken))
		admin.POST("/cache/clear", handler.AdminClearCache(memCache))
		admin.GET("/hidden-users", handler.AdminGetHidden(store, cfg))
		admin.POST("/hidden-users", handler.AdminAddHidden(store))
		admin.DELETE("/hidden-users/:id", handler.AdminRemoveHidden(store))
		admin.GET("/stats", handler.AdminStats(memCache))
	} else {
		log.Println("warn: ADMIN_TOKEN 未设置，/admin/* 路由禁用")
	}

	// 6) 静态资源（embed 自带 dist FS）
	embedpkg.Register(r)

	// 7) 启动 + graceful shutdown
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("listening on :%s (version=%s)", cfg.Port, version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
```

- [ ] **Step 2: 编译验证**

```bash
go build -o /tmp/leaderboard ./cmd/leaderboard
```
Expected: 编译通过

- [ ] **Step 3: 端到端冒烟（用占位前端 + 临时 MySQL）**

```bash
# 临时启 MySQL 容器
docker run -d --rm --name lb_mysql -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=newapi_test -p 3306:3306 mysql:8.0
# 等待 MySQL 就绪（约 20-30s）
sleep 30
mysql -h127.0.0.1 -uroot -proot newapi_test < test/fixtures/seed.sql

# 启动应用
export MYSQL_DSN="root:root@tcp(localhost:3306)/newapi_test?parseTime=true&multiStatements=true"
export ADMIN_TOKEN=test123
export PORT=8080
/tmp/leaderboard &
APP_PID=$!
sleep 2

# 探活
curl -fsS http://localhost:8080/api/health | head
# meta
curl -fsS http://localhost:8080/api/meta | head -c 200; echo
# 土豪榜
curl -fsS "http://localhost:8080/api/leaderboard/rich?range=all&page=1&page_size=3" | head -c 400; echo
# 占位首页
curl -fsS http://localhost:8080/ | head -2

kill $APP_PID 2>/dev/null
docker stop lb_mysql
```

Expected: 4 个请求全部返回正常 JSON / HTML（首页是占位文本，等 Phase 5 出来真前端）

- [ ] **Step 4: 提交**

```bash
git add cmd/
git commit -m "feat(main): wire up all components + graceful shutdown"
```

---

## Phase 5：前端脚手架 + 共用组件

### Task 5.1：前端项目脚手架（Vite + React + TS + Tailwind + 三入口）

**Files:**
- Create: `web/package.json`
- Create: `web/pnpm-lock.yaml`（由 pnpm install 生成）
- Create: `web/vite.config.ts`
- Create: `web/tsconfig.json`
- Create: `web/tsconfig.node.json`
- Create: `web/tailwind.config.ts`
- Create: `web/postcss.config.cjs`
- Create: `web/index.html`
- Create: `web/embed.html`
- Create: `web/admin.html`
- Create: `web/src/main.tsx`、`web/src/embed.tsx`、`web/src/admin.tsx`（最小占位）
- Create: `web/src/styles/globals.css`
- Create: `web/.eslintrc.cjs`

- [ ] **Step 1: 初始化 pnpm 项目**

```bash
cd web
corepack enable
pnpm init
```

- [ ] **Step 2: 安装依赖**

```bash
pnpm add react@^18.3.1 react-dom@^18.3.1 react-router-dom@^6.26.0 \
        @tanstack/react-query@^5.51.0
pnpm add -D vite@^5.4.0 @vitejs/plugin-react@^4.3.0 typescript@^5.5.0 \
        @types/react@^18.3.0 @types/react-dom@^18.3.0 \
        tailwindcss@^3.4.0 postcss@^8.4.0 autoprefixer@^10.4.0 \
        vitest@^2.0.0 @testing-library/react@^16.0.0 @testing-library/jest-dom@^6.4.0 \
        jsdom@^25.0.0 msw@^2.4.0
```

- [ ] **Step 3: 写 `web/package.json` 关键字段**

```json
{
  "name": "newapi-leaderboard-web",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build && node scripts/copy-to-embed.mjs",
    "preview": "vite preview",
    "test": "vitest run",
    "lint": "eslint src --ext ts,tsx"
  }
}
```

- [ ] **Step 4: 写 `web/vite.config.ts`**

```ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'node:path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
  },
  build: {
    target: 'es2020',
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, 'index.html'),
        embed: path.resolve(__dirname, 'embed.html'),
        admin: path.resolve(__dirname, 'admin.html'),
      },
    },
    assetsInlineLimit: 1024,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/admin': 'http://localhost:8080',
    },
  },
})
```

- [ ] **Step 5: 写 `web/tsconfig.json` + `web/tsconfig.node.json`**

`tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "Bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": { "@/*": ["src/*"] },
    "types": ["vitest/globals", "@testing-library/jest-dom"]
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```
`tsconfig.node.json`:
```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "allowSyntheticDefaultImports": true
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 6: 写 `web/tailwind.config.ts`**

```ts
import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './embed.html', './admin.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: { primary: '#8b5cf6', secondary: '#ec4899' },
        gold: '#fbbf24', silver: '#a1a1aa', bronze: '#d97706',
      },
      backdropBlur: { xs: '4px' },
      fontFamily: {
        sans: ['ui-sans-serif', 'system-ui', '-apple-system', 'Segoe UI', 'PingFang SC', 'sans-serif'],
        mono: ['ui-monospace', 'SF Mono', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [],
} satisfies Config
```
`postcss.config.cjs`:
```js
module.exports = { plugins: { tailwindcss: {}, autoprefixer: {} } }
```

- [ ] **Step 7: 写三个 HTML 入口**

`web/index.html`:
```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>NewAPI 排行榜</title>
</head>
<body class="bg-zinc-50 text-zinc-900 antialiased">
  <div id="root"></div>
  <script type="module" src="/src/main.tsx"></script>
</body>
</html>
```
`web/embed.html`:
```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <meta http-equiv="X-Frame-Options" content="ALLOWALL" />
  <title>排行榜</title>
</head>
<body class="bg-transparent">
  <div id="root"></div>
  <script type="module" src="/src/embed.tsx"></script>
</body>
</html>
```
`web/admin.html`:
```html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>排行榜后台</title>
</head>
<body class="bg-zinc-100 text-zinc-900 antialiased">
  <div id="root"></div>
  <script type="module" src="/src/admin.tsx"></script>
</body>
</html>
```

- [ ] **Step 8: 写最小占位入口**

`web/src/main.tsx`:
```tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/globals.css'
createRoot(document.getElementById('root')!).render(
  <StrictMode><div className="p-8">主站脚手架 OK，等待组件接入</div></StrictMode>
)
```
`web/src/embed.tsx`:
```tsx
import { createRoot } from 'react-dom/client'
import './styles/globals.css'
createRoot(document.getElementById('root')!).render(<div className="p-4">嵌入版脚手架 OK</div>)
```
`web/src/admin.tsx`:
```tsx
import { createRoot } from 'react-dom/client'
import './styles/globals.css'
createRoot(document.getElementById('root')!).render(<div className="p-8">后台脚手架 OK</div>)
```

- [ ] **Step 9: 写最小 globals.css**

`web/src/styles/globals.css`:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;

:root {
  --brand-primary: #8b5cf6;
  --brand-secondary: #ec4899;
  --bg-gradient: radial-gradient(circle at 20% 0%, rgba(139,92,246,0.15), transparent 50%),
                  radial-gradient(circle at 80% 100%, rgba(236,72,153,0.12), transparent 50%);
}

@media (max-width: 768px), (hover: none) {
  .glass { backdrop-filter: none !important; background: rgba(255,255,255,0.92) !important; }
}

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

- [ ] **Step 10: 写 `scripts/copy-to-embed.mjs`**（build 后拷贝产物到 `internal/embed/dist/`）

`web/scripts/copy-to-embed.mjs`:
```js
import { cp, mkdir, rm } from 'node:fs/promises'
import { existsSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const src = path.resolve(__dirname, '..', 'dist')
const dest = path.resolve(__dirname, '..', '..', 'internal', 'embed', 'dist')

if (!existsSync(src)) {
  console.error('web/dist 不存在，先 pnpm build')
  process.exit(1)
}
await rm(dest, { recursive: true, force: true })
await mkdir(dest, { recursive: true })
await cp(src, dest, { recursive: true })
// 重新放回 .gitkeep（保证 go embed 总能找到目录）
await cp(path.join(__dirname, '..', '..', '.gitkeep-template'), path.join(dest, '.gitkeep'),
  { force: true }).catch(() => {})
console.log(`copied ${src} → ${dest}`)
```

> 不需要 .gitkeep-template，最后 .catch 可省略。简化：

简化版 `web/scripts/copy-to-embed.mjs`:
```js
import { cp, mkdir, rm, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const src = path.resolve(__dirname, '..', 'dist')
const dest = path.resolve(__dirname, '..', '..', 'internal', 'embed', 'dist')

await rm(dest, { recursive: true, force: true })
await mkdir(dest, { recursive: true })
await cp(src, dest, { recursive: true })
await writeFile(path.join(dest, '.gitkeep'), '')
console.log(`copied ${src} → ${dest}`)
```

- [ ] **Step 11: 跑构建 + 提交**

```bash
cd web
pnpm install
pnpm build
ls ../internal/embed/dist/   # 应该看到 index.html embed.html admin.html assets/
cd ..
go build ./...               # 验证 embed 拿到真实产物
git add web/ internal/embed/dist/.gitkeep
git commit -m "feat(web): vite + react + ts + tailwind scaffolding with 3 entries"
```

---

### Task 5.2：api/client.ts + 类型

**Files:**
- Create: `web/src/api/client.ts`
- Create: `web/src/api/types.ts`
- Test: `web/src/api/__tests__/client.test.ts`

- [ ] **Step 1: 写 types.ts**

`web/src/api/types.ts`:
```ts
export type LeaderboardType =
  | 'rich' | 'spender' | 'topup' | 'foodie' | 'loyal'
  | 'gourmet' | 'biteking' | 'tokens' | 'nightowl'

export type RangeType = 'today' | '7d' | '30d' | 'all'

export interface LeaderboardItem {
  rank: number
  user_id: number
  name: string
  value: number
  value_display: string
  extra?: { model?: string; total_calls?: number } | null
}

export interface LeaderboardResponse {
  type: LeaderboardType
  range: RangeType
  total: number
  page: number
  page_size: number
  list: LeaderboardItem[]
  updated_at: number
  cached: boolean
  stale?: boolean
}

export interface LeaderboardMeta {
  type: LeaderboardType
  name: string
  emoji: string
  description: string
  ranges: RangeType[]
}

export interface MetaResponse {
  leaderboards: LeaderboardMeta[]
  embed: { tabs: LeaderboardType[]; site_url: string; site_name: string }
  version: string
}

export interface PersonalEntry { rank: number; value?: number }

export interface PersonalRank {
  user_id: number
  name: string
  ranks: Partial<Record<LeaderboardType, PersonalEntry>>
}

export interface RankResponse {
  found: boolean
  keyword: string
  results?: PersonalRank[]
}

export interface ApiEnvelope<T> {
  code: number
  msg?: string
  data: T
}
```

- [ ] **Step 2: 写 client.ts**

`web/src/api/client.ts`:
```ts
import type {
  ApiEnvelope, LeaderboardResponse, LeaderboardType,
  MetaResponse, RangeType, RankResponse,
} from './types'

const BASE = ''  // 同源

class ApiError extends Error {
  constructor(public code: number, msg: string) { super(msg) }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    ...init,
    headers: { 'Accept': 'application/json', ...(init?.headers || {}) },
  })
  if (!res.ok) {
    let msg = res.statusText
    try {
      const env = await res.json() as ApiEnvelope<unknown>
      msg = env.msg || msg
    } catch {}
    throw new ApiError(res.status, msg)
  }
  const env = await res.json() as ApiEnvelope<T>
  if (env.code !== 0) throw new ApiError(env.code, env.msg || 'unknown')
  return env.data
}

export const api = {
  meta: () => request<MetaResponse>('/api/meta'),
  leaderboard: (type: LeaderboardType, range: RangeType, page = 1, pageSize = 20) =>
    request<LeaderboardResponse>(
      `/api/leaderboard/${type}?range=${range}&page=${page}&page_size=${pageSize}`),
  rank: (keyword: string) =>
    request<RankResponse>(`/api/rank/${encodeURIComponent(keyword)}`),
  health: () => request<{ db: boolean; version: string }>('/api/health'),
}

export { ApiError }
```

- [ ] **Step 3: 写测试（MSW mock）**

`web/src/api/__tests__/client.test.ts`:
```ts
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'
import { api, ApiError } from '../client'

const server = setupServer(
  http.get('/api/meta', () => HttpResponse.json({
    code: 0, data: { leaderboards: [], embed: { tabs: [], site_url: '', site_name: 'Test' }, version: 'x' },
  })),
  http.get('/api/leaderboard/rich', () => HttpResponse.json({
    code: 0, data: { type: 'rich', range: 'all', total: 1, page: 1, page_size: 5,
      list: [{ rank: 1, user_id: 2, name: 'x', value: 100, value_display: '$100' }],
      updated_at: 0, cached: true },
  })),
  http.get('/api/leaderboard/bad', () => HttpResponse.json({ code: 400, msg: 'invalid type', data: null }, { status: 400 })),
)

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('api client', () => {
  it('fetches meta', async () => {
    const m = await api.meta()
    expect(m.embed.site_name).toBe('Test')
  })

  it('fetches leaderboard', async () => {
    const r = await api.leaderboard('rich', 'all', 1, 5)
    expect(r.list[0].name).toBe('x')
  })

  it('throws ApiError on 400', async () => {
    await expect(api.leaderboard('bad' as any, 'all')).rejects.toBeInstanceOf(ApiError)
  })
})
```

- [ ] **Step 4: 加 vitest 配置**

`web/vitest.config.ts`:
```ts
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
  },
})
```

`web/src/test-setup.ts`:
```ts
import '@testing-library/jest-dom/vitest'
```

- [ ] **Step 5: 跑测试 + 提交**

```bash
cd web
pnpm test
cd ..
git add web/src/api/ web/vitest.config.ts web/src/test-setup.ts
git commit -m "feat(web): api client + types + msw-based unit tests"
```

---

### Task 5.3：共用组件 Avatar / RankRow / TabBar / RangeSwitcher / 状态组件

**Files:**
- Create: `web/src/components/Avatar.tsx`
- Create: `web/src/components/RankRow.tsx`
- Create: `web/src/components/TabBar.tsx`
- Create: `web/src/components/RangeSwitcher.tsx`
- Create: `web/src/components/EmptyState.tsx`
- Create: `web/src/components/ErrorState.tsx`
- Create: `web/src/components/LoadingSkeleton.tsx`
- Create: `web/src/styles/glass.css`
- Test: `web/src/components/__tests__/{Avatar,RankRow,TabBar}.test.tsx`

- [ ] **Step 1: 写 `Avatar.tsx`**

```tsx
import { useMemo } from 'react'

function avatarStyle(userId: number) {
  const hue = (userId * 137.508) % 360
  return {
    background: `linear-gradient(135deg, hsl(${hue}, 70%, 60%), hsl(${(hue+40)%360}, 70%, 50%))`,
  }
}

export function Avatar({ userId, name, size = 32 }: { userId: number; name: string; size?: number }) {
  const style = useMemo(() => ({
    ...avatarStyle(userId),
    width: size, height: size,
    fontSize: Math.max(11, size * 0.4),
  }), [userId, size])
  const letter = (name?.trim()?.[0] ?? '?').toUpperCase()
  return (
    <div
      role="img"
      aria-label={name}
      className="flex-shrink-0 rounded-full flex items-center justify-center text-white font-semibold select-none"
      style={style}
    >{letter}</div>
  )
}
```

- [ ] **Step 2: 写 `glass.css`**

`web/src/styles/glass.css`:
```css
.glass {
  background: rgba(255,255,255,0.75);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255,255,255,0.8);
}

.glass-row { background: rgba(255,255,255,0.7); }
.glass-row-gold   { background: linear-gradient(90deg, rgba(251,191,36,0.25), rgba(255,255,255,0.75)); }
.glass-row-silver { background: linear-gradient(90deg, rgba(212,212,216,0.35), rgba(255,255,255,0.75)); }
.glass-row-bronze { background: linear-gradient(90deg, rgba(217,119,6,0.22), rgba(255,255,255,0.75)); }

.gradient-text {
  background: linear-gradient(135deg, var(--brand-primary), var(--brand-secondary));
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.page-gradient { background-image: var(--bg-gradient); }
```

在 `globals.css` 末尾 `@import './glass.css';`。

- [ ] **Step 3: 写 `RankRow.tsx`**

```tsx
import { Avatar } from './Avatar'
import type { LeaderboardItem } from '@/api/types'

function rowClass(rank: number) {
  if (rank === 1) return 'glass-row-gold'
  if (rank === 2) return 'glass-row-silver'
  if (rank === 3) return 'glass-row-bronze'
  return 'glass-row'
}

function rankIcon(rank: number) {
  if (rank === 1) return '🥇'
  if (rank === 2) return '🥈'
  if (rank === 3) return '🥉'
  return String(rank).padStart(2, '0')
}

export function RankRow({ item, compact = false }: { item: LeaderboardItem; compact?: boolean }) {
  const padY = compact ? 'py-1.5' : 'py-2.5'
  return (
    <div
      className={`flex items-center gap-3 px-3 ${padY} rounded-xl ${rowClass(item.rank)} transition-transform duration-150 hover:scale-[1.01]`}
      style={{ contentVisibility: 'auto', containIntrinsicSize: compact ? '40px' : '56px' }}
    >
      <div className="w-7 text-center font-bold text-zinc-800 text-sm tabular-nums">
        {rankIcon(item.rank)}
      </div>
      <Avatar userId={item.user_id} name={item.name} size={compact ? 24 : 30} />
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-zinc-900 truncate">{item.name}</div>
        {item.extra?.model && (
          <div className="text-[10px] text-zinc-500 truncate">{item.extra.model}</div>
        )}
      </div>
      <div className="text-sm font-bold gradient-text tabular-nums whitespace-nowrap">
        {item.value_display}
      </div>
    </div>
  )
}
```

`web/src/components/__tests__/RankRow.test.tsx`:
```tsx
import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { RankRow } from '../RankRow'

describe('RankRow', () => {
  it('renders gold for rank 1', () => {
    render(<RankRow item={{ rank: 1, user_id: 1, name: 'A', value: 1, value_display: '$1' }} />)
    expect(screen.getByText('🥇')).toBeInTheDocument()
    expect(screen.getByText('A')).toBeInTheDocument()
    expect(screen.getByText('$1')).toBeInTheDocument()
  })
  it('renders extra model when present', () => {
    render(<RankRow item={{ rank: 5, user_id: 5, name: 'B', value: 1, value_display: '$1', extra: { model: 'gpt-4o' } }} />)
    expect(screen.getByText('gpt-4o')).toBeInTheDocument()
  })
})
```

- [ ] **Step 4: 写 `TabBar.tsx`**

```tsx
import type { LeaderboardMeta, LeaderboardType } from '@/api/types'

export function TabBar({ tabs, current, onChange }: {
  tabs: LeaderboardMeta[]
  current: LeaderboardType
  onChange: (t: LeaderboardType) => void
}) {
  return (
    <div className="flex gap-1 overflow-x-auto pb-2 -mx-1 px-1 snap-x">
      {tabs.map(t => {
        const active = t.type === current
        return (
          <button
            key={t.type}
            onClick={() => onChange(t.type)}
            className={`snap-start whitespace-nowrap px-3 py-1.5 rounded-lg text-sm font-semibold transition-colors ${
              active
                ? 'bg-gradient-to-br from-brand-primary to-brand-secondary text-white shadow'
                : 'bg-white/60 text-zinc-600 hover:bg-white border border-zinc-200/60'
            }`}
            aria-pressed={active}
          >
            <span className="mr-1">{t.emoji}</span>{t.name}
          </button>
        )
      })}
    </div>
  )
}
```

- [ ] **Step 5: 写 `RangeSwitcher.tsx`**

```tsx
import type { RangeType } from '@/api/types'

const LABELS: Record<RangeType, string> = {
  today: '今日', '7d': '7 天', '30d': '30 天', all: '全部',
}

export function RangeSwitcher({ ranges, current, onChange }: {
  ranges: RangeType[]; current: RangeType; onChange: (r: RangeType) => void
}) {
  return (
    <div className="inline-flex rounded-lg bg-white/50 border border-zinc-200/60 p-0.5">
      {ranges.map(r => {
        const active = r === current
        return (
          <button
            key={r}
            onClick={() => onChange(r)}
            className={`px-3 py-1 text-xs font-semibold rounded-md transition-colors ${
              active ? 'bg-white shadow text-zinc-900' : 'text-zinc-500 hover:text-zinc-700'
            }`}
            aria-pressed={active}
          >{LABELS[r]}</button>
        )
      })}
    </div>
  )
}
```

- [ ] **Step 6: 写状态组件**

`EmptyState.tsx`:
```tsx
export function EmptyState({ message = '暂无数据' }: { message?: string }) {
  return (
    <div className="py-12 text-center text-zinc-400 text-sm">
      <div className="text-4xl mb-2 opacity-40">📭</div>
      {message}
    </div>
  )
}
```
`ErrorState.tsx`:
```tsx
export function ErrorState({ onRetry, message = '数据加载失败，请稍后再试' }: {
  onRetry?: () => void; message?: string
}) {
  return (
    <div className="py-12 text-center">
      <div className="text-4xl mb-2 opacity-50">⚠️</div>
      <div className="text-sm text-zinc-600 mb-3">{message}</div>
      {onRetry && (
        <button onClick={onRetry}
          className="text-xs px-3 py-1.5 rounded-md bg-brand-primary text-white hover:bg-brand-primary/90">
          重试
        </button>
      )}
    </div>
  )
}
```
`LoadingSkeleton.tsx`:
```tsx
export function LoadingSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="space-y-2 py-2">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="h-10 rounded-xl bg-zinc-200/50 animate-pulse" />
      ))}
    </div>
  )
}
```

- [ ] **Step 7: 跑测试 + 提交**

```bash
cd web
pnpm test
cd ..
git add web/src/components/ web/src/styles/glass.css
git commit -m "feat(web): shared components (Avatar/RankRow/TabBar/RangeSwitcher/states)"
```

---

### Task 5.4：hooks（useLeaderboard / useUrlState）+ PersonalRankWidget

**Files:**
- Create: `web/src/hooks/useLeaderboard.ts`
- Create: `web/src/hooks/useUrlState.ts`
- Create: `web/src/hooks/useMeta.ts`
- Create: `web/src/components/PersonalRankWidget.tsx`
- Test: `web/src/hooks/__tests__/useUrlState.test.ts`

- [ ] **Step 1: 写 `useLeaderboard.ts`**

```ts
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'
import type { LeaderboardType, RangeType } from '@/api/types'

export function useLeaderboard(
  type: LeaderboardType, range: RangeType, page = 1, pageSize = 20, enabled = true,
) {
  return useQuery({
    queryKey: ['lb', type, range, page, pageSize],
    queryFn: () => api.leaderboard(type, range, page, pageSize),
    staleTime: 60_000,
    refetchInterval: 60_000,
    retry: 3,
    enabled,
  })
}
```

- [ ] **Step 2: 写 `useMeta.ts`**

```ts
import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

export function useMeta() {
  return useQuery({
    queryKey: ['meta'],
    queryFn: api.meta,
    staleTime: 10 * 60_000,
  })
}
```

- [ ] **Step 3: 写 `useUrlState.ts`**

```ts
import { useCallback, useEffect, useState } from 'react'

export function useUrlState<T extends Record<string, string>>(defaults: T): [T, (patch: Partial<T>) => void] {
  const [state, setState] = useState<T>(() => readFromUrl(defaults))

  useEffect(() => {
    const onPop = () => setState(readFromUrl(defaults))
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [defaults])

  const update = useCallback((patch: Partial<T>) => {
    setState(prev => {
      const next = { ...prev, ...patch }
      const url = new URL(window.location.href)
      Object.entries(next).forEach(([k, v]) => {
        if (v == null || v === '') url.searchParams.delete(k)
        else url.searchParams.set(k, String(v))
      })
      window.history.pushState({}, '', url)
      return next
    })
  }, [])

  return [state, update]
}

function readFromUrl<T extends Record<string, string>>(defaults: T): T {
  const params = new URLSearchParams(window.location.search)
  const out = { ...defaults }
  for (const k of Object.keys(defaults)) {
    const v = params.get(k)
    if (v != null) (out as any)[k] = v
  }
  return out
}
```

`web/src/hooks/__tests__/useUrlState.test.ts`:
```ts
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, beforeEach } from 'vitest'
import { useUrlState } from '../useUrlState'

describe('useUrlState', () => {
  beforeEach(() => { window.history.pushState({}, '', '/') })

  it('reads initial from URL', () => {
    window.history.pushState({}, '', '/?tab=foodie')
    const { result } = renderHook(() => useUrlState({ tab: 'rich', range: '7d' }))
    expect(result.current[0].tab).toBe('foodie')
  })

  it('updates URL on patch', () => {
    const { result } = renderHook(() => useUrlState({ tab: 'rich' }))
    act(() => result.current[1]({ tab: 'foodie' }))
    expect(window.location.search).toContain('tab=foodie')
  })
})
```

- [ ] **Step 4: 写 `PersonalRankWidget.tsx`**

```tsx
import { useState } from 'react'
import { api } from '@/api/client'
import type { PersonalRank } from '@/api/types'

export function PersonalRankWidget() {
  const [keyword, setKeyword] = useState('')
  const [loading, setLoading] = useState(false)
  const [results, setResults] = useState<PersonalRank[] | null>(null)
  const [notFound, setNotFound] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!keyword.trim()) return
    setLoading(true); setError(null); setResults(null); setNotFound(false)
    try {
      const data = await api.rank(keyword.trim())
      if (!data.found) setNotFound(true)
      else setResults(data.results ?? [])
    } catch (err: any) {
      setError(err.message || '查询失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="glass rounded-xl p-4">
      <form onSubmit={onSubmit} className="flex gap-2">
        <input
          value={keyword} onChange={e => setKeyword(e.target.value)}
          placeholder="输入用户名或昵称查我的排名"
          className="flex-1 px-3 py-1.5 rounded-md bg-white/80 border border-zinc-200 text-sm focus:outline-none focus:ring-2 focus:ring-brand-primary/40"
          maxLength={50}
        />
        <button disabled={loading || !keyword.trim()}
          className="px-3 py-1.5 rounded-md bg-brand-primary text-white text-sm font-semibold disabled:opacity-50">
          {loading ? '查询中…' : '查我在哪'}
        </button>
      </form>
      {error && <div className="mt-3 text-xs text-red-600">{error}</div>}
      {notFound && <div className="mt-3 text-xs text-zinc-500">没找到匹配的用户</div>}
      {results && results.map(r => (
        <div key={r.user_id} className="mt-3 text-sm">
          <div className="font-semibold mb-1">{r.name} <span className="text-xs text-zinc-400">#{r.user_id}</span></div>
          <div className="flex flex-wrap gap-2">
            {Object.entries(r.ranks).length === 0 && (
              <span className="text-xs text-zinc-400">该用户还没上任何榜</span>
            )}
            {Object.entries(r.ranks).map(([type, entry]) => (
              <span key={type} className="text-xs px-2 py-1 rounded-md bg-white/80 border border-zinc-200">
                {type} · #{entry.rank}
              </span>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
```

- [ ] **Step 5: 跑测试 + 提交**

```bash
cd web && pnpm test && cd ..
git add web/src/hooks/ web/src/components/PersonalRankWidget.tsx
git commit -m "feat(web): hooks + PersonalRankWidget"
```

---

## Phase 6：主站 / 嵌入版 / 后台三个入口

### Task 6.1：主站 MainApp（完整版）

**Files:**
- Create: `web/src/pages/MainApp.tsx`
- Create: `web/src/components/LeaderboardCard.tsx`
- Create: `web/src/components/Pagination.tsx`
- Modify: `web/src/main.tsx`

- [ ] **Step 1: 写 `LeaderboardCard.tsx`**

```tsx
import { RankRow } from './RankRow'
import { LoadingSkeleton } from './LoadingSkeleton'
import { ErrorState } from './ErrorState'
import { EmptyState } from './EmptyState'
import type { LeaderboardResponse } from '@/api/types'

export function LeaderboardCard({
  title, emoji, description, data, isLoading, isError, onRetry, compact = false,
}: {
  title: string; emoji: string; description?: string
  data?: LeaderboardResponse
  isLoading: boolean; isError: boolean
  onRetry?: () => void
  compact?: boolean
}) {
  return (
    <div className="glass rounded-2xl p-4">
      <header className="mb-3">
        <h2 className="text-base font-bold gradient-text"><span className="mr-1">{emoji}</span>{title}</h2>
        {description && <p className="text-xs text-zinc-500 mt-0.5">{description}</p>}
        {data?.stale && <p className="text-[10px] text-amber-600 mt-0.5">⚠ 数据可能稍有滞后</p>}
      </header>
      {isLoading && <LoadingSkeleton rows={compact ? 5 : 10} />}
      {isError && <ErrorState onRetry={onRetry} />}
      {!isLoading && !isError && data && (
        data.list.length === 0
          ? <EmptyState message="暂无数据" />
          : <div className="space-y-1.5">
              {data.list.map(item => <RankRow key={item.user_id} item={item} compact={compact} />)}
            </div>
      )}
    </div>
  )
}
```

- [ ] **Step 2: 写 `Pagination.tsx`**

```tsx
export function Pagination({ page, pageSize, total, onChange }: {
  page: number; pageSize: number; total: number; onChange: (p: number) => void
}) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  if (totalPages <= 1) return null
  return (
    <div className="flex items-center justify-center gap-2 mt-4 text-sm">
      <button disabled={page <= 1} onClick={() => onChange(page - 1)}
        className="px-3 py-1 rounded-md bg-white/70 border border-zinc-200 disabled:opacity-40 hover:bg-white">上一页</button>
      <span className="text-zinc-600">第 {page} / {totalPages} 页</span>
      <button disabled={page >= totalPages} onClick={() => onChange(page + 1)}
        className="px-3 py-1 rounded-md bg-white/70 border border-zinc-200 disabled:opacity-40 hover:bg-white">下一页</button>
    </div>
  )
}
```

- [ ] **Step 3: 写 `MainApp.tsx`**

```tsx
import { useMeta } from '@/hooks/useMeta'
import { useLeaderboard } from '@/hooks/useLeaderboard'
import { useUrlState } from '@/hooks/useUrlState'
import { TabBar } from '@/components/TabBar'
import { RangeSwitcher } from '@/components/RangeSwitcher'
import { LeaderboardCard } from '@/components/LeaderboardCard'
import { Pagination } from '@/components/Pagination'
import { PersonalRankWidget } from '@/components/PersonalRankWidget'
import type { LeaderboardType, RangeType } from '@/api/types'

export function MainApp() {
  const meta = useMeta()
  const [state, setState] = useUrlState({ tab: 'rich', range: '7d', page: '1' })

  const currentTab = state.tab as LeaderboardType
  const currentRange = state.range as RangeType
  const currentPage = parseInt(state.page, 10) || 1

  const tabMeta = meta.data?.leaderboards.find(t => t.type === currentTab)
  const supportedRanges = tabMeta?.ranges ?? ['7d']
  const effectiveRange = supportedRanges.includes(currentRange) ? currentRange : supportedRanges[0]

  const lb = useLeaderboard(currentTab, effectiveRange, currentPage, 20, !!tabMeta)
  const siteName = meta.data?.embed.site_name ?? 'NewAPI 排行榜'

  return (
    <div className="page-gradient min-h-screen">
      <div className="max-w-3xl mx-auto px-4 py-8">
        <header className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <h1 className="text-2xl font-bold gradient-text">🏆 {siteName}</h1>
          <div className="sm:w-72"><PersonalRankWidget /></div>
        </header>

        {meta.data && (
          <TabBar
            tabs={meta.data.leaderboards}
            current={currentTab}
            onChange={t => setState({ tab: t, page: '1' })}
          />
        )}

        {tabMeta && (
          <div className="my-3 flex items-center justify-between">
            <RangeSwitcher
              ranges={supportedRanges}
              current={effectiveRange}
              onChange={r => setState({ range: r, page: '1' })}
            />
            {lb.data && (
              <span className="text-xs text-zinc-500">
                更新于 {timeAgo(lb.data.updated_at)}
              </span>
            )}
          </div>
        )}

        {tabMeta && (
          <LeaderboardCard
            title={tabMeta.name}
            emoji={tabMeta.emoji}
            description={tabMeta.description}
            data={lb.data}
            isLoading={lb.isLoading}
            isError={lb.isError}
            onRetry={() => lb.refetch()}
          />
        )}

        {lb.data && (
          <Pagination
            page={currentPage}
            pageSize={lb.data.page_size}
            total={lb.data.total}
            onChange={p => setState({ page: String(p) })}
          />
        )}

        <footer className="mt-12 text-center text-xs text-zinc-400">
          基于 NewAPI · 数据每分钟更新
        </footer>
      </div>
    </div>
  )
}

function timeAgo(ts: number) {
  const diff = Math.max(0, Math.floor(Date.now() / 1000 - ts))
  if (diff < 60) return '刚刚'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`
  return `${Math.floor(diff / 86400)} 天前`
}
```

- [ ] **Step 4: 改 `main.tsx`**

```tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MainApp } from './pages/MainApp'
import './styles/globals.css'

const qc = new QueryClient({
  defaultOptions: { queries: { refetchOnWindowFocus: false } },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={qc}>
      <MainApp />
    </QueryClientProvider>
  </StrictMode>,
)
```

- [ ] **Step 5: alias `@` 已在 Task 5.1 配好（vite.config.ts + tsconfig.json paths），无需改动**

- [ ] **Step 6: 构建 + 手动验证**

```bash
cd web && pnpm build && cd ..
go run ./cmd/leaderboard
# 浏览器打开 http://localhost:8080
```
Expected: 主站可正常切换 9 个 Tab、时间维度、分页；个人查询能搜出 "克吹"

- [ ] **Step 7: 提交**

```bash
git add web/
git commit -m "feat(web): main app with tabs / range / personal rank / pagination"
```

---

### Task 6.2：嵌入版 EmbedApp

**Files:**
- Create: `web/src/pages/EmbedApp.tsx`
- Modify: `web/src/embed.tsx`

- [ ] **Step 1: 写 `EmbedApp.tsx`**

```tsx
import { useState, useEffect, useMemo } from 'react'
import { useMeta } from '@/hooks/useMeta'
import { useLeaderboard } from '@/hooks/useLeaderboard'
import { TabBar } from '@/components/TabBar'
import { LeaderboardCard } from '@/components/LeaderboardCard'
import type { LeaderboardType, RangeType } from '@/api/types'

function readEmbedParams() {
  const p = new URLSearchParams(window.location.search)
  return {
    tabs: (p.get('tabs') || '').split(',').filter(Boolean) as LeaderboardType[],
    range: (p.get('range') || '7d') as RangeType,
    limit: Math.min(parseInt(p.get('limit') || '5', 10) || 5, 10),
    theme: p.get('theme') as 'light' | 'dark' | null,
    site: p.get('site') || '',
  }
}

export function EmbedApp() {
  const params = useMemo(readEmbedParams, [])
  const meta = useMeta()
  const allTabs = meta.data?.leaderboards ?? []
  const tabsToShow = params.tabs.length > 0
    ? allTabs.filter(t => params.tabs.includes(t.type))
    : allTabs.filter(t => (meta.data?.embed.tabs ?? []).includes(t.type))

  const [current, setCurrent] = useState<LeaderboardType>('rich')
  useEffect(() => {
    if (tabsToShow.length > 0) setCurrent(prev =>
      tabsToShow.find(t => t.type === prev) ? prev : tabsToShow[0].type)
  }, [tabsToShow])

  const tabMeta = tabsToShow.find(t => t.type === current)
  const supportedRanges = tabMeta?.ranges ?? ['7d']
  const effectiveRange = supportedRanges.includes(params.range) ? params.range : supportedRanges[0]
  const lb = useLeaderboard(current, effectiveRange, 1, params.limit, !!tabMeta)
  const siteUrl = params.site || meta.data?.embed.site_url || ''

  useEffect(() => {
    if (params.theme === 'dark') document.documentElement.classList.add('dark')
  }, [params.theme])

  return (
    <div className="page-gradient min-h-screen p-3">
      {tabsToShow.length > 0 && (
        <TabBar tabs={tabsToShow} current={current} onChange={setCurrent} />
      )}
      <div className="mt-2">
        {tabMeta && (
          <LeaderboardCard
            title={tabMeta.name}
            emoji={tabMeta.emoji}
            data={lb.data}
            isLoading={lb.isLoading}
            isError={lb.isError}
            onRetry={() => lb.refetch()}
            compact
          />
        )}
      </div>
      {siteUrl && (
        <div className="mt-3 text-center">
          <a href={siteUrl} target="_top"
            className="text-xs text-brand-primary font-semibold hover:underline">
            查看完整版排行榜 →
          </a>
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 2: 改 `embed.tsx`**

```tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { EmbedApp } from './pages/EmbedApp'
import './styles/globals.css'

const qc = new QueryClient({
  defaultOptions: { queries: { refetchOnWindowFocus: false, retry: 2 } },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={qc}>
      <EmbedApp />
    </QueryClientProvider>
  </StrictMode>,
)
```

- [ ] **Step 3: 构建 + 手动验证**

```bash
cd web && pnpm build && cd ..
go run ./cmd/leaderboard
# 浏览器打开 http://localhost:8080/embed?tabs=rich,foodie,nightowl&range=7d&limit=5
# 测试用 newapi 嵌入：
cat > /tmp/test-embed.html <<'EOF'
<!DOCTYPE html><html><body>
<h1>模拟 newapi 首页</h1>
<iframe src="http://localhost:8080/embed?tabs=rich,foodie,nightowl" width="100%" height="500" frameborder="0"></iframe>
</body></html>
EOF
# 用浏览器打开 /tmp/test-embed.html
```
Expected: iframe 内嵌入正常，无 X-Frame-Options 拦截

- [ ] **Step 4: 提交**

```bash
git add web/src/
git commit -m "feat(web): embed app with URL params (tabs/range/limit/theme/site)"
```

---

### Task 6.3：后台 AdminApp

**Files:**
- Create: `web/src/pages/AdminApp.tsx`
- Create: `web/src/api/adminClient.ts`
- Modify: `web/src/admin.tsx`

- [ ] **Step 1: 写 `adminClient.ts`**

```ts
const TOKEN_KEY = 'admin_token'

function token() { return localStorage.getItem(TOKEN_KEY) ?? '' }
function setToken(t: string) { localStorage.setItem(TOKEN_KEY, t) }
function clearToken() { localStorage.removeItem(TOKEN_KEY) }

async function authed<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: {
      Authorization: 'Bearer ' + token(),
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
  })
  if (res.status === 401) { clearToken(); throw new Error('未授权，请重新登录') }
  const env = await res.json()
  if (env.code !== 0) throw new Error(env.msg || '请求失败')
  return env.data
}

export const adminApi = {
  token, setToken, clearToken,
  stats: () => authed<{ cache_hits: number; cache_misses: number; cache_size: number; hit_rate: number }>('/admin/stats'),
  clearCache: (prefix = '') => authed<{ cleared: number }>(`/admin/cache/clear?prefix=${encodeURIComponent(prefix)}`, { method: 'POST' }),
  getHidden: () => authed<{ env: number[]; admin: number[] }>('/admin/hidden-users'),
  addHidden: (userId: number) => authed('/admin/hidden-users', { method: 'POST', body: JSON.stringify({ user_id: userId }) }),
  removeHidden: (userId: number) => authed(`/admin/hidden-users/${userId}`, { method: 'DELETE' }),
}
```

- [ ] **Step 2: 写 `AdminApp.tsx`**

```tsx
import { useEffect, useState } from 'react'
import { adminApi } from '@/api/adminClient'

export function AdminApp() {
  const [authed, setAuthed] = useState(!!adminApi.token())
  if (!authed) return <LoginForm onLogin={() => setAuthed(true)} />
  return <Dashboard onLogout={() => { adminApi.clearToken(); setAuthed(false) }} />
}

function LoginForm({ onLogin }: { onLogin: () => void }) {
  const [t, setT] = useState('')
  const [err, setErr] = useState<string | null>(null)
  async function submit(e: React.FormEvent) {
    e.preventDefault()
    adminApi.setToken(t)
    try {
      await adminApi.stats()
      onLogin()
    } catch (e: any) {
      setErr(e.message)
    }
  }
  return (
    <div className="min-h-screen flex items-center justify-center page-gradient">
      <form onSubmit={submit} className="glass rounded-2xl p-6 w-80 space-y-3">
        <h1 className="text-lg font-bold gradient-text">🛠 后台登录</h1>
        <input type="password" value={t} onChange={e => setT(e.target.value)}
          placeholder="输入 ADMIN_TOKEN"
          className="w-full px-3 py-2 rounded-md bg-white border border-zinc-200 text-sm" />
        {err && <div className="text-xs text-red-600">{err}</div>}
        <button className="w-full py-2 rounded-md bg-brand-primary text-white text-sm font-semibold">登录</button>
      </form>
    </div>
  )
}

function Dashboard({ onLogout }: { onLogout: () => void }) {
  const [stats, setStats] = useState<any>(null)
  const [hidden, setHidden] = useState<{ env: number[]; admin: number[] }>({ env: [], admin: [] })
  const [newId, setNewId] = useState('')
  const [msg, setMsg] = useState<string | null>(null)

  async function refresh() {
    setStats(await adminApi.stats())
    setHidden(await adminApi.getHidden())
  }
  useEffect(() => { refresh() }, [])

  async function doClear() {
    const { cleared } = await adminApi.clearCache()
    setMsg(`已清空缓存（${cleared} 项）`)
    refresh()
  }
  async function doAdd() {
    const id = parseInt(newId, 10)
    if (!id) return
    await adminApi.addHidden(id)
    setNewId('')
    refresh()
  }
  async function doRemove(id: number) {
    await adminApi.removeHidden(id)
    refresh()
  }

  return (
    <div className="min-h-screen page-gradient p-6">
      <div className="max-w-2xl mx-auto space-y-4">
        <header className="flex items-center justify-between">
          <h1 className="text-xl font-bold gradient-text">🛠 排行榜后台</h1>
          <button onClick={onLogout} className="text-xs px-3 py-1 rounded-md bg-white/70 border">退出</button>
        </header>

        {msg && <div className="glass rounded-md p-2 text-sm text-emerald-700">{msg}</div>}

        <section className="glass rounded-2xl p-4">
          <h2 className="font-semibold mb-2">系统状态</h2>
          {stats ? (
            <div className="grid grid-cols-3 gap-3 text-sm">
              <Metric label="缓存命中率" value={`${(stats.hit_rate * 100).toFixed(1)}%`} />
              <Metric label="命中次数" value={String(stats.cache_hits)} />
              <Metric label="缓存项" value={String(stats.cache_size)} />
            </div>
          ) : '加载中...'}
        </section>

        <section className="glass rounded-2xl p-4">
          <h2 className="font-semibold mb-2">缓存管理</h2>
          <button onClick={doClear}
            className="px-3 py-1.5 text-sm rounded-md bg-brand-primary text-white">清空全部</button>
        </section>

        <section className="glass rounded-2xl p-4">
          <h2 className="font-semibold mb-3">隐藏用户</h2>
          <div className="text-xs text-zinc-500 mb-2">
            环境变量隐藏（只读）：{hidden.env.join(', ') || '无'}
          </div>
          <div className="flex gap-2 mb-3">
            <input value={newId} onChange={e => setNewId(e.target.value)}
              placeholder="输入 user_id"
              className="flex-1 px-3 py-1.5 rounded-md bg-white border border-zinc-200 text-sm" />
            <button onClick={doAdd}
              className="px-3 py-1.5 rounded-md bg-brand-primary text-white text-sm font-semibold">+ 添加</button>
          </div>
          <div className="flex flex-wrap gap-2">
            {hidden.admin.length === 0 && <span className="text-xs text-zinc-400">暂无临时隐藏</span>}
            {hidden.admin.map(id => (
              <span key={id} className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-white border border-zinc-200 text-xs">
                {id}
                <button onClick={() => doRemove(id)} className="text-red-500 hover:text-red-700">×</button>
              </span>
            ))}
          </div>
        </section>
      </div>
    </div>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md bg-white/70 p-2 text-center">
      <div className="text-xs text-zinc-500">{label}</div>
      <div className="text-lg font-bold gradient-text">{value}</div>
    </div>
  )
}
```

- [ ] **Step 3: 改 `admin.tsx`**

```tsx
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { AdminApp } from './pages/AdminApp'
import './styles/globals.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode><AdminApp /></StrictMode>,
)
```

- [ ] **Step 4: 构建 + 验证**

```bash
cd web && pnpm build && cd ..
ADMIN_TOKEN=test go run ./cmd/leaderboard
# 浏览器：http://localhost:8080/admin → 输入 "test" → 看到面板
```
Expected: 登录成功 → 看到状态/缓存/隐藏面板，操作生效

- [ ] **Step 5: 提交**

```bash
git add web/src/
git commit -m "feat(web): admin dashboard (login + stats + cache + hidden users)"
```

---

## Phase 7：Docker / CI / 部署文档 / 验收

### Task 7.1：多阶段 Dockerfile

**Files:**
- Create: `Dockerfile`
- Create: `.dockerignore`

- [ ] **Step 1: 写 `.dockerignore`**

```
.git
.gitignore
.github
.vscode
.idea
data/
*.md
docs/
test/
.superpowers/
web/node_modules/
web/dist/
internal/embed/dist/
.env
.env.local
```

- [ ] **Step 2: 写 `Dockerfile`**

```dockerfile
# Stage 1: 前端打包
FROM node:20-alpine AS frontend
WORKDIR /app
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build
# 把 dist 拷到一个标准位置供下一阶段拿
RUN ls -la dist/ && cp -r dist /out-dist

# Stage 2: Go 编译
FROM golang:1.22-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 把前端产物放到 internal/embed/dist 让 go:embed 能找到
COPY --from=frontend /out-dist ./internal/embed/dist
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo docker)" \
    -o /out/leaderboard ./cmd/leaderboard

# Stage 3: 运行时
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone
WORKDIR /app
COPY --from=backend /out/leaderboard .
RUN mkdir -p /app/data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
  CMD wget -q --spider http://127.0.0.1:8080/api/health || exit 1
ENTRYPOINT ["/app/leaderboard"]
```

- [ ] **Step 3: 本地构建镜像验证**

```bash
docker build -t newapi-leaderboard:dev .
docker images | grep newapi-leaderboard   # 镜像应该 ~20-30MB
```

- [ ] **Step 4: 提交**

```bash
git add Dockerfile .dockerignore
git commit -m "build: multi-stage Dockerfile (web + go + alpine runtime)"
```

---

### Task 7.2：docker-compose、zeabur.toml、.env.example、Makefile

**Files:**
- Create: `docker-compose.yml`
- Create: `zeabur.toml`
- Create: `.env.example`
- Create: `Makefile`

- [ ] **Step 1: `.env.example`**

```bash
# ==== 必填 ====
MYSQL_DSN=lb_ro:STRONG_PASSWORD@tcp(mysql.zeabur.internal:3306)/newapi?parseTime=true&loc=Asia%2FShanghai
ADMIN_TOKEN=please-change-me-to-a-long-random-string

# ==== 常用可选 ====
PORT=8080
SITE_NAME=NewAPI 排行榜
SITE_URL=https://leaderboard.example.com
EMBED_TABS_DEFAULT=rich,foodie,nightowl
HIDDEN_USER_IDS=

# ==== 缓存 ====
CACHE_TTL_USERS=60
CACHE_TTL_LOGS=300

# ==== 死忠粉阈值 ====
LOYAL_THRESHOLD=0.8
LOYAL_MIN_CALLS=10

# ==== 高级 ====
MYSQL_MAX_OPEN_CONNS=10
MYSQL_MAX_IDLE_CONNS=5
MYSQL_CONN_MAX_LIFETIME=300
RATE_LIMIT_PER_MIN=200
LOG_LEVEL=info
```

- [ ] **Step 2: `docker-compose.yml`（本地开发）**

```yaml
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: newapi_test
    ports: ["3306:3306"]
    volumes:
      - ./test/fixtures/seed.sql:/docker-entrypoint-initdb.d/01-seed.sql:ro

  app:
    build: .
    depends_on: [mysql]
    environment:
      MYSQL_DSN: "root:root@tcp(mysql:3306)/newapi_test?parseTime=true&loc=Asia%2FShanghai"
      ADMIN_TOKEN: dev-token
      SITE_NAME: "NewAPI 排行榜 (DEV)"
      SITE_URL: "http://localhost:8080"
    ports: ["8080:8080"]
    volumes:
      - ./data:/app/data
```

- [ ] **Step 3: `zeabur.toml`**

```toml
[build]
builder = "dockerfile"

[serve]
port = 8080
healthcheck = "/api/health"
```

- [ ] **Step 4: `Makefile`**

```makefile
.PHONY: dev web build test test-int docker run clean

dev: web
	go run ./cmd/leaderboard

web:
	cd web && pnpm install && pnpm build

build: web
	go build -o bin/leaderboard ./cmd/leaderboard

test:
	go test ./internal/... -race
	cd web && pnpm test

test-int:
	go test -tags=integration ./test/integration/... -v

docker:
	docker build -t newapi-leaderboard:local .

run: docker
	docker run --rm -p 8080:8080 \
		-e MYSQL_DSN="$$MYSQL_DSN" \
		-e ADMIN_TOKEN=test \
		newapi-leaderboard:local

clean:
	rm -rf bin web/dist internal/embed/dist/* internal/embed/dist/.gitkeep
	touch internal/embed/dist/.gitkeep
```

- [ ] **Step 5: 本地用 compose 端到端跑通**

```bash
docker compose up --build
# 浏览器：http://localhost:8080
# 验证嵌入、admin（token=dev-token）
docker compose down
```

- [ ] **Step 6: 提交**

```bash
git add docker-compose.yml zeabur.toml .env.example Makefile
git commit -m "build: docker-compose, zeabur config, env example, makefile"
```

---

### Task 7.3：README 部署文档

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 写完整 README**

`README.md`:
```markdown
# NewAPI 排行榜

> 给 NewAPI 用户做的社区向外部排行榜。9 个有趣榜单（土豪/吃货/熬夜冠军…），支持 iframe 嵌入到 NewAPI 首页，部署到 Zeabur。

[![CI](https://github.com/yourname/newapi-leaderboard/actions/workflows/ci.yml/badge.svg)](https://github.com/yourname/newapi-leaderboard/actions)

## 截图

（部署后补）

## 特性

- 💰 9 个核心榜单（土豪 / 散财 / 充值 / 吃货 / 死忠粉 / 美食家 / 单笔王 / 吞噬 / 熬夜冠军）
- 🍴 完整版主站 + 紧凑嵌入版 widget（iframe 任意域名）
- 🛠 轻量后台管理面板（缓存清理、临时隐藏用户、状态查看）
- ⚡ 内存缓存 + singleflight 防击穿，对 NewAPI 主库压力极小
- 🔒 只读直连数据库，强制使用只读账号
- 🐳 单 Docker 镜像（~25MB），Zeabur 一键部署

## Zeabur 一图流部署

```
NewAPI 已部署 ─→ 同项目 + Service ─→ Docker (本仓库)
                                    │
                                    ├ MYSQL_DSN = readonly 账号 + mysql.zeabur.internal
                                    ├ ADMIN_TOKEN = 强随机串
                                    └ 绑域名: leaderboard.your.com
```

### 步骤

1. **建只读 MySQL 账号**（在 NewAPI 的 MySQL 实例执行）：
   ```sql
   CREATE USER 'lb_ro'@'%' IDENTIFIED BY '强密码';
   GRANT SELECT ON newapi.* TO 'lb_ro'@'%';
   FLUSH PRIVILEGES;
   ```
2. Zeabur 控制台 → New Project / 已有项目 → **Add Service** → **Git** → 选本仓库
3. 自动识别 Dockerfile 构建
4. 设置环境变量（最少 `MYSQL_DSN` + `ADMIN_TOKEN`，参考 `.env.example`）
5. 绑定持久卷到 `/app/data`（存临时隐藏用户列表）
6. 绑定子域名（如 `leaderboard.your.com`）
7. 在 NewAPI 后台首页添加：
   ```html
   <iframe src="https://leaderboard.your.com/embed?tabs=rich,foodie,nightowl"
           width="100%" height="420" frameborder="0" loading="lazy"
           style="border-radius:14px"></iframe>
   ```

## 嵌入版 URL 参数

| 参数 | 默认 | 说明 |
|---|---|---|
| `tabs` | `EMBED_TABS_DEFAULT` | 逗号分隔，显示哪几个 Tab |
| `range` | `7d` | 默认时间窗 |
| `limit` | `5` | 每榜显示行数（上限 10） |
| `theme` | 系统跟随 | `light` / `dark` |
| `site` | `SITE_URL` | "查看完整版"跳转地址 |

## 环境变量

参考 [`.env.example`](.env.example)。

## 本地开发

```bash
# 用 docker-compose 起 MySQL + 应用
docker compose up --build

# 或分开跑（前端开发热更新）
docker run -d --rm -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=newapi_test \
  -v $(pwd)/test/fixtures/seed.sql:/docker-entrypoint-initdb.d/seed.sql \
  --name lb_mysql mysql:8.0

# 后端
export MYSQL_DSN="root:root@tcp(localhost:3306)/newapi_test?parseTime=true"
export ADMIN_TOKEN=dev
go run ./cmd/leaderboard       # 8080

# 前端
cd web && pnpm install && pnpm dev   # 5173 (proxy /api → 8080)
```

## 测试

```bash
# 单元测试
make test

# 集成测试（需 Docker，会启临时 MySQL）
make test-int
```

## 安全

- **强制只读账号**：`MYSQL_DSN` 仅授予 `SELECT` 权限，杜绝误写
- **接口白名单**：仅返回 `user_id / name / value / extra`，不泄露 email / token 等
- **Admin 鉴权**：`crypto/subtle.ConstantTimeCompare` 比对 token
- **限流**：公开 API 200 req/min/IP
- **任意嵌入**：CORS `*` + `Content-Security-Policy: frame-ancestors *`（如需限制改 middleware/cors.go）

## 设计文档与实施计划

- 设计：[`docs/superpowers/specs/2026-05-24-newapi-leaderboard-design.md`](docs/superpowers/specs/2026-05-24-newapi-leaderboard-design.md)
- 计划：[`docs/superpowers/plans/2026-05-24-newapi-leaderboard.md`](docs/superpowers/plans/2026-05-24-newapi-leaderboard.md)

## License

MIT
```

- [ ] **Step 2: 提交**

```bash
git add README.md
git commit -m "docs: comprehensive README with zeabur deployment guide"
```

---

### Task 7.4：GitHub Actions CI

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/docker.yml`

- [ ] **Step 1: 写 `ci.yml`**

```yaml
name: CI
on:
  push: { branches: [main] }
  pull_request: { branches: [main] }

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_ROOT_PASSWORD: root
          MYSQL_DATABASE: newapi_test
        ports: [3306:3306]
        options: --health-cmd="mysqladmin ping" --health-interval=10s --health-timeout=5s --health-retries=10
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - uses: pnpm/action-setup@v4
        with: { version: 9 }
      - uses: actions/setup-node@v4
        with: { node-version: "20", cache: "pnpm", cache-dependency-path: web/pnpm-lock.yaml }

      - name: Frontend install + build
        working-directory: web
        run: |
          pnpm install --frozen-lockfile
          pnpm test
          pnpm build

      - name: Go unit tests
        run: go test ./internal/... -race -coverprofile=cover.out

      - name: Go integration tests
        env:
          MYSQL_HOST: 127.0.0.1
          MYSQL_PORT: 3306
        run: go test -tags=integration ./test/integration/... -v -timeout 5m

      - name: Build binary
        run: go build -o /tmp/lb ./cmd/leaderboard

      - name: Docker build smoke test
        run: docker build -t lb:ci .
```

- [ ] **Step 2: 写 `docker.yml`**

```yaml
name: Docker
on:
  push:
    branches: [main]
    tags: ['v*']

permissions:
  contents: read
  packages: write

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-qemu-action@v3
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/metadata-action@v5
        id: meta
        with:
          images: ghcr.io/${{ github.repository_owner }}/newapi-leaderboard
          tags: |
            type=raw,value=latest,enable={{is_default_branch}}
            type=semver,pattern={{version}}
            type=sha,prefix=sha-,format=short
      - uses: docker/build-push-action@v6
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

- [ ] **Step 3: 提交**

```bash
git add .github/
git commit -m "ci: GitHub Actions for tests and multi-arch docker image"
```

---

### Task 7.5：最终验收（端到端冒烟 + 验收清单）

> 这一步不写代码，按 spec §17 的 DoD 逐项验收。如发现某项未达标，补 issue/task 修复。

- [ ] **Step 1: 功能验收**

| 项 | 测试方法 | 通过 |
|---|---|---|
| 9 个榜单全部聚合正确 | `make test-int` 全 PASS | ☐ |
| today/7d/30d/all 全部可用 | 手动切换主站时间维度 | ☐ |
| 个人查询找到 / 未找到 | 输入"克吹" + "不存在的" | ☐ |
| 嵌入版任意域名 iframe | `/tmp/test-embed.html` 用 file:// 打开 | ☐ |
| URL 参数 tabs/range/limit/theme/site 生效 | 改 URL 看反应 | ☐ |
| Admin 鉴权 + 缓存清理 + 临时隐藏 | 后台手动操作 | ☐ |
| HIDDEN_USER_IDS 环境变量 | 设置后 admin 用户消失 | ☐ |
| LOYAL_THRESHOLD / LOYAL_MIN_CALLS | 设 0.99 / 1000 看榜单变化 | ☐ |

- [ ] **Step 2: 质量验收**

| 项 | 通过 |
|---|---|
| README + .env.example 明确标注只读账号要求 | ☐ |
| 9 个榜单 SQL 都有集成测试 | ☐ |
| CI 全绿（go test + pnpm test + docker build） | ☐ |
| 移动端 375px 不破，毛玻璃自动降级 | ☐ |
| API 缓存命中 P95 < 50ms（手动 ab/wrk） | ☐ |
| 缓存未命中（logs 100 万行场景）P95 < 1.5s | ☐ |

- [ ] **Step 3: 交付验收**

| 项 | 通过 |
|---|---|
| `docker compose up --build` 一键启动 | ☐ |
| 推到 GHCR 的镜像可被 Zeabur 拉取部署 | ☐ |
| Zeabur 实际部署后嵌入 NewAPI 首页 | ☐ |

- [ ] **Step 4: 打 v1.0 tag**

```bash
git tag -a v1.0.0 -m "First public release"
git push origin v1.0.0
# GHCR docker.yml 触发，构建 latest 和 v1.0.0 镜像
```

- [ ] **Step 5: 项目交付**

宣告 v1.0 完成。后续 v2 路线图参考 spec §19。

---

**计划结束**

> 全部任务约 25 个 (Phase 0-7)，按 TDD 顺序逐个执行；每个 task 完成即 commit，便于回滚。如遇到 spec 与现实冲突，先更新 spec 再调整 plan，再实施。


