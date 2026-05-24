package integration

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	sharedDB   *sql.DB
	sharedDSN  string
	sharedOnce sync.Once
	sharedErr  error
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
}
