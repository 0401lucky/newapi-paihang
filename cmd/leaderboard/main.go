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

	"github.com/0401lucky/newapi-paihang/internal/cache"
	"github.com/0401lucky/newapi-paihang/internal/config"
	"github.com/0401lucky/newapi-paihang/internal/db"
	embedpkg "github.com/0401lucky/newapi-paihang/internal/embed"
	"github.com/0401lucky/newapi-paihang/internal/handler"
	"github.com/0401lucky/newapi-paihang/internal/middleware"
	"github.com/0401lucky/newapi-paihang/internal/persist"
	"github.com/0401lucky/newapi-paihang/internal/service"
)

var version = "dev"

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

	api := r.Group("/api", middleware.RateLimit(cfg.RateLimitPerMin))
	api.GET("/meta", handler.Meta(cfg, version))
	api.GET("/leaderboard/:type", handler.Leaderboard(svc, store))
	api.GET("/rank/:keyword", handler.Rank(svc, store))
	api.GET("/health", handler.Health(database, version))

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

	// 6) 静态资源
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
