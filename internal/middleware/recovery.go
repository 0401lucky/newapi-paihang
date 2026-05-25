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
				log.Printf("PANIC %s %s: %v\n%s",
					c.Request.Method, c.Request.URL.Path, r, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code": 500, "msg": "服务内部错误",
				})
			}
		}()
		c.Next()
	}
}
