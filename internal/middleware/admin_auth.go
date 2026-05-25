package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminAuth 用 ADMIN_TOKEN 鉴权 /admin/* 路由。Token 为空时直接返回 404。
func AdminAuth(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expectedToken == "" {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"code": 404, "msg": "admin not configured",
			})
			return
		}
		got := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(expectedToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401, "msg": "unauthorized",
			})
			return
		}
		c.Next()
	}
}
