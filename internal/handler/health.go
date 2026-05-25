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
			"data": gin.H{"db": dbOK, "version": version},
		})
	}
}
