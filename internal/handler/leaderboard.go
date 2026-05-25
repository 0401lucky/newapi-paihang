package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/0401lucky/newapi-paihang/internal/persist"
	"github.com/0401lucky/newapi-paihang/internal/service"
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
