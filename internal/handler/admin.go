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
