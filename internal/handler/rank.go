package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/0401lucky/newapi-paihang/internal/persist"
	"github.com/0401lucky/newapi-paihang/internal/service"
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
